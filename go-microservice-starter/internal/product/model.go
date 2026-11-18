package product

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const QueueStockChanged = "inventory.stock_changed"

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidPayload  = errors.New("invalid payload: check fields and ensure price/stock >= 0")
	ErrNegativeStock   = errors.New("stock cannot be negative")
	ErrVersionConflict = errors.New("conflict: product was modified by another request")
	ErrSKUConflict     = errors.New("SKU might already exist or database error")
)

// Product โครงสร้างตารางสินค้าสำหรับ GORM
type Product struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	SKU       string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"sku"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Price     float64        `gorm:"type:numeric(12,2);not null" json:"price"`
	Stock     int            `gorm:"not null;default:0" json:"stock"`
	Version   int            `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// CreateProductRequest Struct สำหรับตรวจสอบค่าตอน Bind Request (POST / PUT)
type CreateProductRequest struct {
	SKU   string  `json:"sku"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// PatchStockRequest Struct สำหรับอัปเดตเฉพาะสต็อก (PATCH)
type PatchStockRequest struct {
	Stock int `json:"stock"`
}

// StockChangedEvent Event Payload สำหรับส่งเข้า RabbitMQ
type StockChangedEvent struct {
	EventID   string    `json:"event_id"`
	ProductID uuid.UUID `json:"product_id"`
	SKU       string    `json:"sku"`
	NewStock  int       `json:"new_stock"`
	Timestamp time.Time `json:"timestamp"`
}
