package database

import (
	"context"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// NewPostgresDB initializes a PostgreSQL connection using sqlx (with pgx/v5 stdlib driver).
func NewPostgresDB(dsn string) (*sqlx.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)

	return db, nil
}

// Ping verifies database connectivity for readiness probes.
func Ping(ctx context.Context, db *sqlx.DB) error {
	return db.PingContext(ctx)
}
