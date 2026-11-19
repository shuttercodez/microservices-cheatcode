package order

import (
	"context"
	"testing"

	"go-microservice-starter/internal/product"

	"github.com/google/uuid"
)

// ==================== 1. Mock Objects for Testing Order Service ====================

// mockOrderRepository simulates an in-memory orders database table.
type mockOrderRepository struct {
	orders map[string]*Order
}

func newMockOrderRepository() *mockOrderRepository {
	return &mockOrderRepository{
		orders: make(map[string]*Order),
	}
}

func (m *mockOrderRepository) Create(ctx context.Context, order *Order) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	m.orders[order.ID.String()] = order
	return nil
}

func (m *mockOrderRepository) FindAll(ctx context.Context, page, limit int) ([]Order, int64, error) {
	var list []Order
	for _, o := range m.orders {
		list = append(list, *o)
	}
	return list, int64(len(list)), nil
}

func (m *mockOrderRepository) FindByID(ctx context.Context, id string) (*Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

type mockOrderCache struct {
	store     map[string]*Order
	listStore map[string]*PaginatedOrders
}

func newMockOrderCache() *mockOrderCache {
	return &mockOrderCache{
		store:     make(map[string]*Order),
		listStore: make(map[string]*PaginatedOrders),
	}
}

func (m *mockOrderCache) Get(ctx context.Context, id string) (*Order, error) {
	o, ok := m.store[id]
	if !ok {
		return nil, nil
	}
	copied := *o
	return &copied, nil
}

func (m *mockOrderCache) Set(ctx context.Context, order *Order) error {
	copied := *order
	m.store[order.ID.String()] = &copied
	return nil
}

func (m *mockOrderCache) GetList(ctx context.Context, page, limit int) (*PaginatedOrders, error) {
	key := listCacheKey(page, limit)
	list, ok := m.listStore[key]
	if !ok {
		return nil, nil
	}
	copied := *list
	return &copied, nil
}

func (m *mockOrderCache) SetList(ctx context.Context, page, limit int, list *PaginatedOrders) error {
	key := listCacheKey(page, limit)
	copied := *list
	m.listStore[key] = &copied
	return nil
}

func (m *mockOrderCache) InvalidateLists(ctx context.Context) error {
	m.listStore = make(map[string]*PaginatedOrders)
	return nil
}

// mockProductService simulates cross-domain calls to ProductService.
type mockProductService struct {
	products         map[string]*product.Product
	patchStockCalled bool
	forcePatchErr    error
}

func newMockProductService() *mockProductService {
	return &mockProductService{
		products: make(map[string]*product.Product),
	}
}

func (m *mockProductService) Create(ctx context.Context, req product.CreateProductRequest) (*product.Product, error) {
	return nil, nil
}

func (m *mockProductService) GetAll(ctx context.Context, page, limit int) (*product.PaginatedProducts, error) {
	return nil, nil
}

func (m *mockProductService) GetByID(ctx context.Context, id string) (*product.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, product.ErrProductNotFound
	}
	copied := *p
	return &copied, nil
}

func (m *mockProductService) Update(ctx context.Context, id string, req product.CreateProductRequest) (*product.Product, error) {
	return nil, nil
}

func (m *mockProductService) PatchStock(ctx context.Context, id string, req product.PatchStockRequest) (*product.Product, error) {
	m.patchStockCalled = true
	if m.forcePatchErr != nil {
		return nil, m.forcePatchErr
	}
	p, ok := m.products[id]
	if !ok {
		return nil, product.ErrProductNotFound
	}
	p.Stock = req.Stock
	p.Version++
	return p, nil
}

func (m *mockProductService) Delete(ctx context.Context, id string) error {
	return nil
}

// mockOrderProducer records published order events in memory for test assertions.
type mockOrderProducer struct {
	publishedOrders []Order
}

func (m *mockOrderProducer) PublishOrderCreated(ctx context.Context, order Order) error {
	m.publishedOrders = append(m.publishedOrders, order)
	return nil
}

// ==================== 2. Unit Tests for Order Service (Cross-Domain) ====================

func TestOrderService_CreateOrder(t *testing.T) {
	productID := uuid.New()

	t.Run("Success - creates order, deducts product stock, calculates total price, and publishes event", func(t *testing.T) {
		// Arrange
		orderRepo := newMockOrderRepository()
		prodSvc := newMockProductService()
		producer := &mockOrderProducer{}

		// Seed product with 10 items in stock at 1,500 per unit
		prodSvc.products[productID.String()] = &product.Product{
			ID:      productID,
			SKU:     "KEY-001",
			Name:    "Mechanical Keyboard",
			Price:   1500,
			Stock:   10,
			Version: 1,
		}

		svc := NewService(orderRepo, newMockOrderCache(), prodSvc, producer)

		// Act: order 3 units
		order, err := svc.CreateOrder(context.Background(), CreateOrderRequest{
			ProductID: productID.String(),
			Quantity:  3,
		})

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 1. Verify total price calculation (1,500 * 3 = 4,500)
		if order.TotalPrice != 4500 {
			t.Errorf("expected total price 4500, got %.2f", order.TotalPrice)
		}
		// 2. Verify product stock was deducted to 7 (10 - 3)
		if prodSvc.products[productID.String()].Stock != 7 {
			t.Errorf("expected remaining product stock 7, got %d", prodSvc.products[productID.String()].Stock)
		}
		// 3. Verify Producer published the `order.created` event
		if len(producer.publishedOrders) != 1 {
			t.Errorf("expected 1 order event published, got %d", len(producer.publishedOrders))
		}
	})

	t.Run("Fail - ordering more than available stock returns ErrInsufficientStock without deducting stock", func(t *testing.T) {
		// Arrange
		orderRepo := newMockOrderRepository()
		prodSvc := newMockProductService()
		producer := &mockOrderProducer{}

		// Seed product with only 2 units remaining
		prodSvc.products[productID.String()] = &product.Product{
			ID:    productID,
			SKU:   "KEY-001",
			Price: 1500,
			Stock: 2,
		}

		svc := NewService(orderRepo, newMockOrderCache(), prodSvc, producer)

		// Act: attempt to order 5 units
		_, err := svc.CreateOrder(context.Background(), CreateOrderRequest{
			ProductID: productID.String(),
			Quantity:  5,
		})

		// Assert
		if err != ErrInsufficientStock {
			t.Fatalf("expected error %v, got %v", ErrInsufficientStock, err)
		}
		if prodSvc.patchStockCalled {
			t.Errorf("expected PatchStock NOT to be called when stock is insufficient")
		}
		if len(producer.publishedOrders) != 0 {
			t.Errorf("expected no order event published, got %d", len(producer.publishedOrders))
		}
	})

	t.Run("Fail - non-existent product returns product.ErrProductNotFound", func(t *testing.T) {
		orderRepo := newMockOrderRepository()
		prodSvc := newMockProductService()
		producer := &mockOrderProducer{}
		svc := NewService(orderRepo, newMockOrderCache(), prodSvc, producer)

		_, err := svc.CreateOrder(context.Background(), CreateOrderRequest{
			ProductID: uuid.NewString(),
			Quantity:  1,
		})

		if err != product.ErrProductNotFound {
			t.Fatalf("expected error %v, got %v", product.ErrProductNotFound, err)
		}
	})
}
