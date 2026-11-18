package product

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository defines the data access contract for product persistence.
type Repository interface {
	Create(ctx context.Context, product *Product) error
	FindAll(ctx context.Context) ([]Product, error)
	FindByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, product *Product) error
	UpdateStockWithVersion(ctx context.Context, product *Product, newStock int) error
	Delete(ctx context.Context, id string) error
}

type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a sqlx-backed Product Repository (Dapper-style struct mapping via `db` tags).
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

func (r *sqlxRepository) Create(ctx context.Context, product *Product) error {
	const query = `
		INSERT INTO products (sku, name, price, stock, version)
		VALUES ($1, $2, $3, $4, 1)
		RETURNING id, sku, name, price, stock, version, created_at, updated_at, deleted_at
	`
	if err := r.db.GetContext(ctx, product, query, product.SKU, product.Name, product.Price, product.Stock); err != nil {
		return ErrSKUConflict
	}
	return nil
}

func (r *sqlxRepository) FindAll(ctx context.Context) ([]Product, error) {
	const query = `
		SELECT id, sku, name, price, stock, version, created_at, updated_at, deleted_at
		FROM products
		WHERE deleted_at IS NULL
	`
	products := make([]Product, 0)
	if err := r.db.SelectContext(ctx, &products, query); err != nil {
		return nil, err
	}
	return products, nil
}

func (r *sqlxRepository) FindByID(ctx context.Context, id string) (*Product, error) {
	const query = `
		SELECT id, sku, name, price, stock, version, created_at, updated_at, deleted_at
		FROM products
		WHERE id = $1 AND deleted_at IS NULL
	`
	var product Product
	if err := r.db.GetContext(ctx, &product, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return &product, nil
}

func (r *sqlxRepository) Update(ctx context.Context, product *Product) error {
	const query = `
		UPDATE products
		SET sku = $1, name = $2, price = $3, stock = $4, version = $5, updated_at = $6
		WHERE id = $7 AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	_, err := r.db.ExecContext(
		ctx,
		query,
		product.SKU,
		product.Name,
		product.Price,
		product.Stock,
		product.Version,
		now,
		product.ID,
	)
	if err != nil {
		return err
	}
	product.UpdatedAt = now
	return nil
}

// UpdateStockWithVersion updates product stock using optimistic locking via the version column.
func (r *sqlxRepository) UpdateStockWithVersion(ctx context.Context, product *Product, newStock int) error {
	const query = `
		UPDATE products
		SET stock = $1, version = $2, updated_at = $3
		WHERE id = $4 AND version = $5 AND deleted_at IS NULL
	`
	now := time.Now().UTC()
	nextVersion := product.Version + 1

	result, err := r.db.ExecContext(ctx, query, newStock, nextVersion, now, product.ID, product.Version)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrVersionConflict
	}

	product.Stock = newStock
	product.Version = nextVersion
	product.UpdatedAt = now
	return nil
}

func (r *sqlxRepository) Delete(ctx context.Context, id string) error {
	const query = `
		UPDATE products
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrProductNotFound
	}
	return nil
}
