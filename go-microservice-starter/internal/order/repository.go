package order

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository defines the data access contract for order persistence.
type Repository interface {
	Create(ctx context.Context, order *Order) error
	FindAll(ctx context.Context, page, limit int) ([]Order, int64, error)
	FindByID(ctx context.Context, id string) (*Order, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates a GORM-backed Order Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, order *Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *gormRepository) FindAll(ctx context.Context, page, limit int) ([]Order, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&Order{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	var orders []Order
	if err := r.db.WithContext(ctx).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *gormRepository) FindByID(ctx context.Context, id string) (*Order, error) {
	var order Order
	if err := r.db.WithContext(ctx).First(&order, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}
