package product

import (
	"context"
)

// Service defines the business logic contract for managing products.
type Service interface {
	Create(ctx context.Context, req CreateProductRequest) (*Product, error)
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, id string, req CreateProductRequest) (*Product, error)
	PatchStock(ctx context.Context, id string, req PatchStockRequest) (*Product, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo     Repository
	cache    Cache
	producer Producer
}

// NewService creates a new Product Service with injected Repository, Cache, and Producer dependencies.
func NewService(repo Repository, cache Cache, producer Producer) Service {
	return &service{
		repo:     repo,
		cache:    cache,
		producer: producer,
	}
}

func (s *service) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	if req.SKU == "" || req.Name == "" || req.Price < 0 || req.Stock < 0 {
		return nil, ErrInvalidPayload
	}

	product := &Product{
		SKU:   req.SKU,
		Name:  req.Name,
		Price: req.Price,
		Stock: req.Stock,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, product)
	}
	_ = s.producer.PublishStockChanged(ctx, *product)
	return product, nil
}

func (s *service) GetAll(ctx context.Context) ([]Product, error) {
	return s.repo.FindAll(ctx)
}

// GetByID implements the Cache-Aside pattern: check Redis first, fallback to DB on miss, and populate Redis.
func (s *service) GetByID(ctx context.Context, id string) (*Product, error) {
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, id); err == nil && cached != nil {
			return cached, nil
		}
	}

	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, product)
	}
	return product, nil
}

func (s *service) Update(ctx context.Context, id string, req CreateProductRequest) (*Product, error) {
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.SKU == "" || req.Name == "" || req.Price < 0 || req.Stock < 0 {
		return nil, ErrInvalidPayload
	}

	product.SKU = req.SKU
	product.Name = req.Name
	product.Price = req.Price
	product.Stock = req.Stock
	product.Version++

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}

	// Update or invalidate cache so subsequent reads get fresh data
	if s.cache != nil {
		_ = s.cache.Set(ctx, product)
	}
	_ = s.producer.PublishStockChanged(ctx, *product)
	return product, nil
}

func (s *service) PatchStock(ctx context.Context, id string, req PatchStockRequest) (*Product, error) {
	if req.Stock < 0 {
		return nil, ErrNegativeStock
	}

	// Always read current version directly from the database for optimistic locking
	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateStockWithVersion(ctx, product, req.Stock); err != nil {
		return nil, err
	}

	// Refresh Redis cache with the new stock and incremented version
	if s.cache != nil {
		_ = s.cache.Set(ctx, product)
	}
	_ = s.producer.PublishStockChanged(ctx, *product)
	return product, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.Delete(ctx, id)
	}
	return nil
}
