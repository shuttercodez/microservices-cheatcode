package product

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go-microservice-starter/internal/platform/queue"

	"github.com/google/uuid"
)

// Service กำหนด Interface สำหรับ Business Logic ของระบบสินค้า
type Service interface {
	Create(ctx context.Context, req CreateProductRequest) (*Product, error)
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, id string, req CreateProductRequest) (*Product, error)
	PatchStock(ctx context.Context, id string, req PatchStockRequest) (*Product, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo Repository
	mq   *queue.RabbitMQ
}

// NewService สร้าง Service พร้อมผูก Repository และ RabbitMQ
func NewService(repo Repository, mq *queue.RabbitMQ) Service {
	return &service{
		repo: repo,
		mq:   mq,
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

	s.publishStockEvent(*product)
	return product, nil
}

func (s *service) GetAll(ctx context.Context) ([]Product, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id string) (*Product, error) {
	return s.repo.FindByID(ctx, id)
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

	s.publishStockEvent(*product)
	return product, nil
}

func (s *service) PatchStock(ctx context.Context, id string, req PatchStockRequest) (*Product, error) {
	if req.Stock < 0 {
		return nil, ErrNegativeStock
	}

	product, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateStockWithVersion(ctx, product, req.Stock); err != nil {
		return nil, err
	}

	s.publishStockEvent(*product)
	return product, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) publishStockEvent(p Product) {
	if s.mq == nil {
		return
	}

	event := StockChangedEvent{
		EventID:   uuid.NewString(),
		ProductID: p.ID,
		SKU:       p.SKU,
		NewStock:  p.Stock,
		Timestamp: time.Now(),
	}
	body, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal stock event", "error", err)
		return
	}

	if err := s.mq.Publish(context.Background(), QueueStockChanged, body); err != nil {
		slog.Error("Failed to publish message to RabbitMQ", "error", err)
	} else {
		slog.Info("Published event to RabbitMQ (Fiber)", "event_id", event.EventID, "sku", p.SKU, "stock", p.Stock)
	}
}
