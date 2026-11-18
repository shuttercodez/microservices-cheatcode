package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgresDB initializes a PostgreSQL connection using GORM.
func NewPostgresDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Ping verifies database connectivity for readiness probes.
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
