package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ manages the AMQP connection and channel for message publishing.
type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// NewRabbitMQ establishes a connection to RabbitMQ and opens a default channel.
func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &RabbitMQ{
		Conn:    conn,
		Channel: ch,
	}, nil
}

// DeclareQueue declares a durable queue on the RabbitMQ broker.
func (r *RabbitMQ) DeclareQueue(queueName string) error {
	_, err := r.Channel.QueueDeclare(queueName, true, false, false, false, nil)
	return err
}

// Publish sends a persistent JSON message to the specified queue.
func (r *RabbitMQ) Publish(ctx context.Context, queueName string, body []byte) error {
	return r.Channel.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // Persist messages to disk to survive broker restarts
		Body:         body,
	})
}

// IsClosed checks whether the underlying AMQP connection is closed.
func (r *RabbitMQ) IsClosed() bool {
	return r.Conn == nil || r.Conn.IsClosed()
}

// Close safely closes the AMQP channel and connection.
func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		_ = r.Channel.Close()
	}
	if r.Conn != nil {
		_ = r.Conn.Close()
	}
}
