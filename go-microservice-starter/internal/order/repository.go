package order

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository defines the data access contract for order persistence.
type Repository interface {
	Create(ctx context.Context, order *Order) error
	FindAll(ctx context.Context) ([]Order, error)
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

func (r *gormRepository) FindAll(ctx context.Context) ([]Order, error) {
	var orders []Order
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
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
