package product

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go-microservice-starter/internal/platform/queue"

	"github.com/google/uuid"
)

// Producer defines the interface for publishing product domain events to a message broker.
type Producer interface {
	PublishStockChanged(ctx context.Context, product Product) error
}

type rabbitMQProducer struct {
	mq *queue.RabbitMQ
}

// NewProducer creates a new RabbitMQ-backed product event producer.
func NewProducer(mq *queue.RabbitMQ) Producer {
	return &rabbitMQProducer{
		mq: mq,
	}
}

// PublishStockChanged constructs a StockChangedEvent, serializes it to JSON, and publishes it to the queue.
func (p *rabbitMQProducer) PublishStockChanged(ctx context.Context, product Product) error {
	if p.mq == nil {
		return nil
	}

	event := StockChangedEvent{
		EventID:   uuid.NewString(),
		ProductID: product.ID,
		SKU:       product.SKU,
		NewStock:  product.Stock,
		Timestamp: time.Now(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal stock event", "error", err)
		return err
	}

	if err := p.mq.Publish(ctx, QueueStockChanged, body); err != nil {
		slog.Error("Failed to publish message to RabbitMQ", "error", err)
		return err
	}

	slog.Info("📤 [Producer Published Event]",
		"event_id", event.EventID,
		"sku", product.SKU,
		"stock", product.Stock,
	)
	return nil
}
