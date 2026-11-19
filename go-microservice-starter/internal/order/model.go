package order

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const QueueOrderCreated = "order.created"

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidOrderInput = errors.New("invalid order payload: product_id is required and quantity must be > 0")
	ErrInsufficientStock = errors.New("insufficient product stock for this order")
)

// Order represents the database entity for customer orders managed by GORM.
type Order struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	Quantity   int            `gorm:"not null" json:"quantity"`
	TotalPrice float64        `gorm:"type:numeric(12,2);not null" json:"total_price"`
	Status     string         `gorm:"type:varchar(32);not null;default:'COMPLETED'" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// PaginatedOrders represents a paginated list response for orders.
type PaginatedOrders struct {
	Data       []Order `json:"data"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	Total      int64   `json:"total"`
	TotalPages int     `json:"total_pages"`
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
