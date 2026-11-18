package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ จัดการ Connection และ Channel สำหรับส่งข้อความ
type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// NewRabbitMQ เชื่อมต่อ RabbitMQ และเปิด Channel พื้นฐาน
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

// DeclareQueue สร้าง Queue แบบ Durable
func (r *RabbitMQ) DeclareQueue(queueName string) error {
	_, err := r.Channel.QueueDeclare(queueName, true, false, false, false, nil)
	return err
}

// Publish ส่งข้อความแบบ Persistent เข้า Queue
func (r *RabbitMQ) Publish(ctx context.Context, queueName string, body []byte) error {
	return r.Channel.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // ป้องกันข้อความหายเมื่อ MQ ล่ม
		Body:         body,
	})
}

// IsClosed เช็กว่า Connection ปิดไปแล้วหรือยัง
func (r *RabbitMQ) IsClosed() bool {
	return r.Conn == nil || r.Conn.IsClosed()
}

// Close ปิด Channel และ Connection อย่างปลอดภัย
func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		_ = r.Channel.Close()
	}
	if r.Conn != nil {
		_ = r.Conn.Close()
	}
}
