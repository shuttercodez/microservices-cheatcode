package order

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const QueueOrderCreated = "order.created"

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidOrderInput = errors.New("invalid order payload: product_id is required and quantity must be > 0")
	ErrInsufficientStock = errors.New("insufficient product stock for this order")
)

// Order represents the database entity for customer orders mapped directly via SQL/pgx (no ORM).
type Order struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	ProductID  uuid.UUID  `db:"product_id" json:"product_id"`
	Quantity   int        `db:"quantity" json:"quantity"`
	TotalPrice float64    `db:"total_price" json:"total_price"`
	Status     string     `db:"status" json:"status"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at" json:"-"`
}

// CreateOrderRequest represents the request DTO for creating a new order (POST /api/v1/orders).
type CreateOrderRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// OrderCreatedEvent represents the event payload published to RabbitMQ when an order is created.
type OrderCreatedEvent struct {
	EventID    string    `json:"event_id"`
	OrderID    uuid.UUID `json:"order_id"`
	ProductID  uuid.UUID `json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Timestamp  time.Time `json:"timestamp"`
}
