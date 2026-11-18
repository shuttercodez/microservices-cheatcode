package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgresDB เชื่อมต่อ PostgreSQL ด้วย GORM
func NewPostgresDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Ping ตรวจสอบสถานะการเชื่อมต่อ Database สำหรับ Readiness Probe
func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
