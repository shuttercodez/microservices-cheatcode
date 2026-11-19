package product

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository defines the data access contract for product persistence.
type Repository interface {
	Create(ctx context.Context, product *Product) error
	FindAll(ctx context.Context, page, limit int) ([]Product, int64, error)
	FindByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, product *Product) error
	UpdateStockWithVersion(ctx context.Context, product *Product, newStock int) error
	Delete(ctx context.Context, id string) error
}

type gormRepository struct {
	db *gorm.DB
}

// NewRepository creates a GORM-backed Product Repository.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, product *Product) error {
	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		return ErrSKUConflict
	}
	return nil
}

func (r *gormRepository) FindAll(ctx context.Context, page, limit int) ([]Product, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&Product{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	var products []Product
	if err := r.db.WithContext(ctx).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *gormRepository) FindByID(ctx context.Context, id string) (*Product, error) {
	var product Product
	if err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return &product, nil
}

func (r *gormRepository) Update(ctx context.Context, product *Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

// UpdateStockWithVersion updates product stock using optimistic locking via the version column.
func (r *gormRepository) UpdateStockWithVersion(ctx context.Context, product *Product, newStock int) error {
	result := r.db.WithContext(ctx).Model(&Product{}).
		Where("id = ? AND version = ?", product.ID, product.Version).
		Updates(map[string]interface{}{
			"stock":   newStock,
			"version": product.Version + 1,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVersionConflict
	}

	product.Stock = newStock
	product.Version++
	return nil
}

func (r *gormRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Product{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}
	return nil
}
