package order

import (
	"context"

	"go-sqlx-microservice-starter/internal/product"
)

// Service defines the business logic contract for managing orders.
type Service interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error)
	GetAllOrders(ctx context.Context, page, limit int) (*PaginatedOrders, error)
	GetOrderByID(ctx context.Context, id string) (*Order, error)
}

type service struct {
	orderRepo      Repository
	orderCache     Cache
	productService product.Service // Cross-domain dependency injected via interface
	producer       Producer
}

// NewService creates an Order Service with injected OrderRepository, OrderCache, ProductService, and OrderProducer.
func NewService(orderRepo Repository, orderCache Cache, productService product.Service, producer Producer) Service {
	return &service{
		orderRepo:      orderRepo,
		orderCache:     orderCache,
		productService: productService,
		producer:       producer,
	}
}

func normalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	return page, limit
}

func calcTotalPages(total int64, limit int) int {
	if total == 0 || limit <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}

// CreateOrder verifies product availability, deducts stock via ProductService, persists the order, caches it, and publishes an event.
func (s *service) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	if req.ProductID == "" || req.Quantity <= 0 {
		return nil, ErrInvalidOrderInput
	}

	// 1. Fetch product details and verify stock availability via ProductService
	prod, err := s.productService.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}

	if prod.Stock < req.Quantity {
		return nil, ErrInsufficientStock
	}

	// 2. Deduct stock via ProductService.PatchStock (enforcing optimistic locking)
	newStock := prod.Stock - req.Quantity
	_, err = s.productService.PatchStock(ctx, req.ProductID, product.PatchStockRequest{
		Stock: newStock,
	})
	if err != nil {
		return nil, err
	}

	// 3. Calculate total price and persist the order record
	order := &Order{
		ProductID:  prod.ID,
		Quantity:   req.Quantity,
		TotalPrice: prod.Price * float64(req.Quantity),
		Status:     "COMPLETED",
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// 4. Populate Redis cache and invalidate cached order lists
	if s.orderCache != nil {
		_ = s.orderCache.Set(ctx, order)
		_ = s.orderCache.InvalidateLists(ctx)
	}

	// 5. Publish order created event to RabbitMQ
	_ = s.producer.PublishOrderCreated(ctx, *order)

	return order, nil
}

// GetAllOrders retrieves a paginated list of orders using Cache-Aside with Redis.
func (s *service) GetAllOrders(ctx context.Context, page, limit int) (*PaginatedOrders, error) {
	page, limit = normalizePagination(page, limit)

	if s.orderCache != nil {
		if cached, err := s.orderCache.GetList(ctx, page, limit); err == nil && cached != nil {
			return cached, nil
		}
	}

	orders, total, err := s.orderRepo.FindAll(ctx, page, limit)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []Order{}
	}

	res := &PaginatedOrders{
		Data:       orders,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: calcTotalPages(total, limit),
	}

	if s.orderCache != nil {
		_ = s.orderCache.SetList(ctx, page, limit, res)
	}
	return res, nil
}

// GetOrderByID retrieves an order by ID using Cache-Aside with Redis.
func (s *service) GetOrderByID(ctx context.Context, id string) (*Order, error) {
	if s.orderCache != nil {
		if cached, err := s.orderCache.Get(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}

	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.orderCache != nil {
		_ = s.orderCache.Set(ctx, order)
	}
	return order, nil
}
