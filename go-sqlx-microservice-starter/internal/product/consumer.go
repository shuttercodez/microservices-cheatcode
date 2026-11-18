package product

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// StartQueueConsumer runs a background worker to consume stock change events from RabbitMQ.
func StartQueueConsumer(ctx context.Context, conn *amqp.Connection, queueName string) {
	ch, err := conn.Channel()
	if err != nil {
		slog.Error("Consumer failed to open channel", "error", err)
		return
	}
	defer ch.Close()

	msgs, err := ch.Consume(queueName, "inventory-worker", false, false, false, false, nil)
	if err != nil {
		slog.Error("Failed to register consumer", "error", err)
		return
	}

	slog.Info("RabbitMQ Consumer started waiting for messages...")
	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping RabbitMQ consumer...")
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			var event StockChangedEvent
			if err := json.Unmarshal(d.Body, &event); err == nil {
				slog.Info("📥 [Fiber Consumer Received Event]",
					"event_id", event.EventID,
					"sku", event.SKU,
					"new_stock", event.NewStock,
				)
				_ = d.Ack(false)
			}
		}
	}
}
