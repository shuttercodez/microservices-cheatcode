package order

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"go-sqlx-microservice-starter/internal/platform/queue"

	"github.com/google/uuid"
)

// Producer defines the interface for publishing order domain events to a message broker.
type Producer interface {
	PublishOrderCreated(ctx context.Context, order Order) error
}

type rabbitMQProducer struct {
	mq *queue.RabbitMQ
}

// NewProducer creates a new RabbitMQ-backed order event producer.
func NewProducer(mq *queue.RabbitMQ) Producer {
	return &rabbitMQProducer{mq: mq}
}

func (p *rabbitMQProducer) PublishOrderCreated(ctx context.Context, order Order) error {
	if p.mq == nil {
		return nil
	}

	event := OrderCreatedEvent{
		EventID:    uuid.NewString(),
		OrderID:    order.ID,
		ProductID:  order.ProductID,
		Quantity:   order.Quantity,
		TotalPrice: order.TotalPrice,
		Timestamp:  time.Now(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal order event", "error", err)
		return err
	}

	if err := p.mq.Publish(ctx, QueueOrderCreated, body); err != nil {
		slog.Error("Failed to publish order event to RabbitMQ", "error", err)
		return err
	}

	slog.Info("📤 [Order Producer Published Event]",
		"event_id", event.EventID,
		"order_id", order.ID,
		"product_id", order.ProductID,
		"total_price", order.TotalPrice,
	)
	return nil
}
