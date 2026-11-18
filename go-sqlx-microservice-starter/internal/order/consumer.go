package order

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// StartQueueConsumer runs a background worker to consume order creation events from RabbitMQ.
func StartQueueConsumer(ctx context.Context, conn *amqp.Connection, queueName string) {
	ch, err := conn.Channel()
	if err != nil {
		slog.Error("Order consumer failed to open channel", "error", err)
		return
	}
	defer ch.Close()

	msgs, err := ch.Consume(queueName, "order-worker", false, false, false, false, nil)
	if err != nil {
		slog.Error("Failed to register order consumer", "error", err)
		return
	}

	slog.Info("RabbitMQ Order Consumer started waiting for messages...")
	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping RabbitMQ order consumer...")
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			var event OrderCreatedEvent
			if err := json.Unmarshal(d.Body, &event); err == nil {
				slog.Info("📥 [Order Consumer Received Event]",
					"event_id", event.EventID,
					"order_id", event.OrderID,
					"product_id", event.ProductID,
					"quantity", event.Quantity,
					"total_price", event.TotalPrice,
				)
				_ = d.Ack(false)
			}
		}
	}
}
