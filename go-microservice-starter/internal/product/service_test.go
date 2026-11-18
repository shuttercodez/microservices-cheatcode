package product

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// ==================== 1. Mock Objects (Equivalent to Moq / Mockito) ====================

// mockRepository simulates an in-memory product database repository.
type mockRepository struct {
	products         map[string]*Product
	findByIDCalls    int
	forceCreateErr   error
	forceConflictErr bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		products: make(map[string]*Product),
	}
}

func (m *mockRepository) Create(ctx context.Context, product *Product) error {
	if m.forceCreateErr != nil {
		return m.forceCreateErr
	}
	if product.ID == uuid.Nil {
		product.ID = uuid.New()
	}
	product.Version = 1
	m.products[product.ID.String()] = product
	return nil
}

func (m *mockRepository) FindAll(ctx context.Context) ([]Product, error) {
	var list []Product
	for _, p := range m.products {
		list = append(list, *p)
	}
	return list, nil
}

func (m *mockRepository) FindByID(ctx context.Context, id string) (*Product, error) {
	m.findByIDCalls++
	p, ok := m.products[id]
	if !ok {
		return nil, ErrProductNotFound
	}
	// Return a copy to simulate fetching a fresh record from the database
	copied := *p
	return &copied, nil
}

func (m *mockRepository) Update(ctx context.Context, product *Product) error {
	m.products[product.ID.String()] = product
	return nil
}

func (m *mockRepository) UpdateStockWithVersion(ctx context.Context, product *Product, newStock int) error {
	if m.forceConflictErr {
		return ErrVersionConflict
	}
	stored, ok := m.products[product.ID.String()]
	if !ok {
		return ErrProductNotFound
	}
	if stored.Version != product.Version {
		return ErrVersionConflict
	}
	stored.Stock = newStock
	stored.Version++

	product.Stock = newStock
	product.Version = stored.Version
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	if _, ok := m.products[id]; !ok {
		return ErrProductNotFound
	}
	delete(m.products, id)
	return nil
}

// mockCache simulates a Redis cache in memory.
type mockCache struct {
	store map[string]*Product
}

func newMockCache() *mockCache {
	return &mockCache{
		store: make(map[string]*Product),
	}
}

func (m *mockCache) Get(ctx context.Context, id string) (*Product, error) {
	p, ok := m.store[id]
	if !ok {
		return nil, nil // Cache miss
	}
	copied := *p
	return &copied, nil
}

func (m *mockCache) Set(ctx context.Context, product *Product) error {
	copied := *product
	m.store[product.ID.String()] = &copied
	return nil
}

func (m *mockCache) Delete(ctx context.Context, id string) error {
	delete(m.store, id)
	return nil
}

// mockProducer records published events in memory for test assertions.
type mockProducer struct {
	publishedEvents []Product
}

func (m *mockProducer) PublishStockChanged(ctx context.Context, product Product) error {
	m.publishedEvents = append(m.publishedEvents, product)
	return nil
}

// ==================== 2. Unit Tests for Product Service ====================

func TestProductService_Create(t *testing.T) {
	tests := []struct {
		name            string
		req             CreateProductRequest
		forceRepoErr    error
		wantErr         error
		wantEventCalled bool
	}{
		{
			name: "Success - creates product, caches it, and publishes event to queue",
			req: CreateProductRequest{
				SKU:   "LAPTOP-001",
				Name:  "Gaming Laptop",
				Price: 35000,
				Stock: 10,
			},
			wantErr:         nil,
			wantEventCalled: true,
		},
		{
			name: "Fail - negative price returns ErrInvalidPayload and does not publish event",
			req: CreateProductRequest{
				SKU:   "LAPTOP-002",
				Name:  "Broken Laptop",
				Price: -100,
				Stock: 5,
			},
			wantErr:         ErrInvalidPayload,
			wantEventCalled: false,
		},
		{
			name: "Fail - duplicate SKU returns ErrSKUConflict",
			req: CreateProductRequest{
				SKU:   "DUP-SKU",
				Name:  "Duplicate Item",
				Price: 500,
				Stock: 1,
			},
			forceRepoErr:    ErrSKUConflict,
			wantErr:         ErrSKUConflict,
			wantEventCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := newMockRepository()
			repo.forceCreateErr = tt.forceRepoErr
			cache := newMockCache()
			producer := &mockProducer{}
			svc := NewService(repo, cache, producer)

			// Act
			res, err := svc.Create(context.Background(), tt.req)

			// Assert
			if err != tt.wantErr {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if res == nil || res.SKU != tt.req.SKU {
					t.Errorf("expected product SKU %s, got %+v", tt.req.SKU, res)
				}
			}

			eventCalled := len(producer.publishedEvents) > 0
			if eventCalled != tt.wantEventCalled {
				t.Errorf("expected producer event called = %v, got %v", tt.wantEventCalled, eventCalled)
			}
		})
	}
}

func TestProductService_CacheAside_GetByID(t *testing.T) {
	// Arrange
	repo := newMockRepository()
	cache := newMockCache()
	producer := &mockProducer{}
	svc := NewService(repo, cache, producer)

	// Seed product directly in repository (simulating an existing DB record not yet in Redis)
	id := uuid.New()
	repo.products[id.String()] = &Product{
		ID:      id,
		SKU:     "CACHE-001",
		Name:    "Cached Item",
		Price:   1200,
		Stock:   5,
		Version: 1,
	}

	// 1st Call: Cache Miss -> should query DB (findByIDCalls == 1) and populate Redis cache
	p1, err := svc.GetByID(context.Background(), id.String())
	if err != nil || p1.SKU != "CACHE-001" {
		t.Fatalf("unexpected error on 1st GetByID: %v", err)
	}
	if repo.findByIDCalls != 1 {
		t.Fatalf("expected 1 DB call on cache miss, got %d", repo.findByIDCalls)
	}

	// 2nd Call: Cache Hit -> should return from Redis WITHOUT querying DB again (findByIDCalls stays 1)
	p2, err := svc.GetByID(context.Background(), id.String())
	if err != nil || p2.SKU != "CACHE-001" {
		t.Fatalf("unexpected error on 2nd GetByID: %v", err)
	}
	if repo.findByIDCalls != 1 {
		t.Errorf("expected DB calls to remain 1 on cache hit, got %d", repo.findByIDCalls)
	}
}

func TestProductService_PatchStock_OptimisticLocking(t *testing.T) {
	// Arrange
	repo := newMockRepository()
	cache := newMockCache()
	producer := &mockProducer{}
	svc := NewService(repo, cache, producer)

	created, _ := svc.Create(context.Background(), CreateProductRequest{
		SKU:   "MOUSE-01",
		Name:  "Wireless Mouse",
		Price: 990,
		Stock: 20,
	})
	// Clear initial creation event before testing PatchStock
	producer.publishedEvents = nil

	t.Run("Success - updates stock, refreshes cache, and increments version from 1 to 2", func(t *testing.T) {
		updated, err := svc.PatchStock(context.Background(), created.ID.String(), PatchStockRequest{Stock: 15})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Stock != 15 {
			t.Errorf("expected stock 15, got %d", updated.Stock)
		}
		if updated.Version != 2 {
			t.Errorf("expected version 2, got %d", updated.Version)
		}
		// Verify Redis cache also has the updated stock (15)
		cached, _ := cache.Get(context.Background(), created.ID.String())
		if cached == nil || cached.Stock != 15 {
			t.Errorf("expected cache to be updated with stock 15, got %+v", cached)
		}
		if len(producer.publishedEvents) != 1 {
			t.Errorf("expected 1 event published, got %d", len(producer.publishedEvents))
		}
	})

	t.Run("Fail - negative stock returns ErrNegativeStock", func(t *testing.T) {
		_, err := svc.PatchStock(context.Background(), created.ID.String(), PatchStockRequest{Stock: -5})
		if err != ErrNegativeStock {
			t.Fatalf("expected %v, got %v", ErrNegativeStock, err)
		}
	})

	t.Run("Fail - concurrent modification returns ErrVersionConflict", func(t *testing.T) {
		repo.forceConflictErr = true
		_, err := svc.PatchStock(context.Background(), created.ID.String(), PatchStockRequest{Stock: 10})
		if err != ErrVersionConflict {
			t.Fatalf("expected %v, got %v", ErrVersionConflict, err)
		}
	})
}
