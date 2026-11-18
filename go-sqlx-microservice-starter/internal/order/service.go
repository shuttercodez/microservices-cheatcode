package order

import (
	"context"

	"go-sqlx-microservice-starter/internal/product"
)

// Service defines the business logic contract for managing orders.
type Service interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error)
	GetAllOrders(ctx context.Context) ([]Order, error)
	GetOrderByID(ctx context.Context, id string) (*Order, error)
}

type service struct {
	orderRepo      Repository
	productService product.Service // Cross-domain dependency injected via interface
	producer       Producer
}

// NewService creates an Order Service with injected OrderRepository, ProductService, and OrderProducer.
func NewService(orderRepo Repository, productService product.Service, producer Producer) Service {
	return &service{
		orderRepo:      orderRepo,
		productService: productService,
		producer:       producer,
	}
}

// CreateOrder verifies product availability, deducts stock via ProductService, persists the order, and publishes an event.
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

	// 4. Publish order created event to RabbitMQ
	_ = s.producer.PublishOrderCreated(ctx, *order)

	return order, nil
}

func (s *service) GetAllOrders(ctx context.Context) ([]Order, error) {
	return s.orderRepo.FindAll(ctx)
}

func (s *service) GetOrderByID(ctx context.Context, id string) (*Order, error) {
	return s.orderRepo.FindByID(ctx, id)
}
