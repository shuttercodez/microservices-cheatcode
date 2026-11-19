package product

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const QueueStockChanged = "inventory.stock_changed"

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidPayload  = errors.New("invalid payload: check fields and ensure price/stock >= 0")
	ErrNegativeStock   = errors.New("stock cannot be negative")
	ErrVersionConflict = errors.New("conflict: product was modified by another request")
	ErrSKUConflict     = errors.New("SKU might already exist or database error")
)

// Product represents the database entity for products mapped directly via SQL/sqlx (no ORM).
type Product struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	SKU       string     `db:"sku" json:"sku"`
	Name      string     `db:"name" json:"name"`
	Price     float64    `db:"price" json:"price"`
	Stock     int        `db:"stock" json:"stock"`
	Version   int        `db:"version" json:"version"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

// PaginatedProducts represents a paginated list response for products.
type PaginatedProducts struct {
	Data       []Product `json:"data"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	Total      int64     `json:"total"`
	TotalPages int       `json:"total_pages"`
}

// CreateProductRequest represents the request DTO for creating or replacing a product (POST / PUT).
type CreateProductRequest struct {
	SKU   string  `json:"sku"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// PatchStockRequest represents the request DTO for updating product stock (PATCH).
type PatchStockRequest struct {
	Stock int `json:"stock"`
}

// StockChangedEvent represents the event payload published to RabbitMQ when product stock changes.
type StockChangedEvent struct {
	EventID   string    `json:"event_id"`
	ProductID uuid.UUID `json:"product_id"`
	SKU       string    `json:"sku"`
	NewStock  int       `json:"new_stock"`
	Timestamp time.Time `json:"timestamp"`
}
