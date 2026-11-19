package order

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// Repository defines the data access contract for order persistence.
type Repository interface {
	Create(ctx context.Context, order *Order) error
	FindAll(ctx context.Context, page, limit int) ([]Order, int64, error)
	FindByID(ctx context.Context, id string) (*Order, error)
}

type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a sqlx-backed Order Repository (Dapper-style struct mapping via `db` tags).
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

func (r *sqlxRepository) Create(ctx context.Context, order *Order) error {
	const query = `
		INSERT INTO orders (product_id, quantity, total_price, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, product_id, quantity, total_price, status, created_at, updated_at, deleted_at
	`
	return r.db.GetContext(
		ctx,
		order,
		query,
		order.ProductID,
		order.Quantity,
		order.TotalPrice,
		order.Status,
	)
}

func (r *sqlxRepository) FindAll(ctx context.Context, page, limit int) ([]Order, int64, error) {
	const countQuery = `SELECT COUNT(*) FROM orders WHERE deleted_at IS NULL`
	var total int64
	if err := r.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, err
	}

	const query = `
		SELECT id, product_id, quantity, total_price, status, created_at, updated_at, deleted_at
		FROM orders
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	offset := (page - 1) * limit
	orders := make([]Order, 0)
	if err := r.db.SelectContext(ctx, &orders, query, limit, offset); err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *sqlxRepository) FindByID(ctx context.Context, id string) (*Order, error) {
	const query = `
		SELECT id, product_id, quantity, total_price, status, created_at, updated_at, deleted_at
		FROM orders
		WHERE id = $1 AND deleted_at IS NULL
	`
	var order Order
	if err := r.db.GetContext(ctx, &order, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}
