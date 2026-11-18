package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ==================== 1. GORM Models & DTOs ====================

type Product struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	SKU       string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"sku" binding:"required"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name" binding:"required"`
	Price     float64        `gorm:"type:numeric(12,2);not null" json:"price" binding:"required,gte=0"`
	Stock     int            `gorm:"not null;default:0" json:"stock" binding:"gte=0"`
	Version   int            `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// DTO สำหรับ PATCH (อัปเดตเฉพาะสต็อก)
type PatchStockRequest struct {
	Stock int `json:"stock" binding:"required,gte=0"`
}

// Event Payload สำหรับส่งเข้า RabbitMQ
type StockChangedEvent struct {
	EventID   string    `json:"event_id"`
	ProductID uuid.UUID `json:"product_id"`
	SKU       string    `json:"sku"`
	NewStock  int       `json:"new_stock"`
	Timestamp time.Time `json:"timestamp"`
}

// ==================== 2. Main Application ====================

func main() {
	// ใช้ Structured JSON Logger ตามมาตรฐาน Microservices
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. เชื่อมต่อ PostgreSQL ด้วย GORM
	dsn := "host=localhost user=appuser password=apppassword dbname=inventory_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}

	// 2. เชื่อมต่อ RabbitMQ และสร้าง Queue
	rmqConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		slog.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer rmqConn.Close()

	rmqCh, err := rmqConn.Channel()
	if err != nil {
		slog.Error("Failed to open a RabbitMQ channel", "error", err)
		os.Exit(1)
	}
	defer rmqCh.Close()

	queueName := "inventory.stock_changed"
	_, err = rmqCh.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		slog.Error("Failed to declare queue", "error", err)
		os.Exit(1)
	}

	// 3. รัน Background Consumer (จำลองฝั่งรับ Message จาก Queue)
	ctx, cancelConsumer := context.WithCancel(context.Background())
	go startQueueConsumer(ctx, rmqConn, queueName)

	// 4. ตั้งค่า HTTP Router (Gin)
	r := gin.New()
	r.Use(gin.Recovery())

	// --- Microservice Health & Readiness Probes ---
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil || rmqConn.IsClosed() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// --- RESTful API Endpoints (ครบทุก HTTP Methods) ---
	v1 := r.Group("/api/v1/products")
	{
		// POST: สร้างสินค้าใหม่
		v1.POST("", func(c *gin.Context) {
			var req Product
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := db.WithContext(c.Request.Context()).Create(&req).Error; err != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "SKU might already exist or DB error"})
				return
			}
			publishStockEvent(rmqCh, queueName, req)
			c.JSON(http.StatusCreated, req)
		})

		// GET (All): ดึงรายการสินค้าทั้งหมด
		v1.GET("", func(c *gin.Context) {
			var products []Product
			if err := db.WithContext(c.Request.Context()).Find(&products).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, products)
		})

		// GET (By ID): ดูสินค้าตาม ID
		v1.GET("/:id", func(c *gin.Context) {
			var product Product
			if err := db.WithContext(c.Request.Context()).First(&product, "id = ?", c.Param("id")).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, product)
		})

		// PUT: แก้ไขข้อมูลสินค้าทั้งก้อน (Full Replace)
		v1.PUT("/:id", func(c *gin.Context) {
			var product Product
			if err := db.First(&product, "id = ?", c.Param("id")).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}

			var req Product
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			product.SKU = req.SKU
			product.Name = req.Name
			product.Price = req.Price
			product.Stock = req.Stock
			product.Version++

			if err := db.WithContext(c.Request.Context()).Save(&product).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			publishStockEvent(rmqCh, queueName, product)
			c.JSON(http.StatusOK, product)
		})

		// PATCH: อัปเดตเฉพาะจำนวนสต็อก (Partial Update + Optimistic Locking)
		v1.PATCH("/:id/stock", func(c *gin.Context) {
			var req PatchStockRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			var product Product
			if err := db.First(&product, "id = ?", c.Param("id")).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}

			// อัปเดตพร้อมเช็ก Version เพื่อป้องกัน Race Condition
			result := db.WithContext(c.Request.Context()).Model(&Product{}).
				Where("id = ? AND version = ?", product.ID, product.Version).
				Updates(map[string]interface{}{
					"stock":   req.Stock,
					"version": product.Version + 1,
				})

			if result.RowsAffected == 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "conflict: product was modified by another request"})
				return
			}

			product.Stock = req.Stock
			product.Version++
			publishStockEvent(rmqCh, queueName, product)
			c.JSON(http.StatusOK, product)
		})

		// DELETE: ลบสินค้า (Soft Delete ด้วย GORM)
		v1.DELETE("/:id", func(c *gin.Context) {
			result := db.WithContext(c.Request.Context()).Delete(&Product{}, "id = ?", c.Param("id"))
			if result.RowsAffected == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}
			c.Status(http.StatusNoContent)
		})
	}

	// 5. รัน HTTP Server พร้อมระบบ Graceful Shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		slog.Info("Starting HTTP server on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed", "error", err)
		}
	}()

	// รอรับสัญญาณปิดโปรแกรม (Ctrl+C หรือ Kubernetes SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down service gracefully...")
	cancelConsumer() // สั่งหยุด Consumer

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	slog.Info("Service exited cleanly")
}

// ==================== 3. RabbitMQ Producer & Consumer ====================

func publishStockEvent(ch *amqp.Channel, queueName string, p Product) {
	event := StockChangedEvent{
		EventID:   uuid.NewString(),
		ProductID: p.ID,
		SKU:       p.SKU,
		NewStock:  p.Stock,
		Timestamp: time.Now(),
	}
	body, _ := json.Marshal(event)

	err := ch.PublishWithContext(context.Background(), "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // เก็บลงดิสก์ป้องกันข้อความหายถ้า RabbitMQ รีสตาร์ท
		Body:         body,
	})
	if err != nil {
		slog.Error("Failed to publish message to RabbitMQ", "error", err)
	} else {
		slog.Info("Published event to RabbitMQ", "event_id", event.EventID, "sku", p.SKU, "stock", p.Stock)
	}
}

func startQueueConsumer(ctx context.Context, conn *amqp.Connection, queueName string) {
	ch, err := conn.Channel()
	if err != nil {
		slog.Error("Consumer failed to open channel", "error", err)
		return
	}
	defer ch.Close()

	msgs, err := ch.Consume(queueName, "inventory-worker", false, false, false, false, nil)
	if err != nil {
		slog.Error("Failed to register consumer", "error", err)
		return
	}

	slog.Info("RabbitMQ Consumer started waiting for messages...")
	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping RabbitMQ consumer...")
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			var event StockChangedEvent
			if err := json.Unmarshal(d.Body, &event); err == nil {
				slog.Info("📥 [Consumer Received Event]",
					"event_id", event.EventID,
					"sku", event.SKU,
					"new_stock", event.NewStock,
				)
				// ส่ง ACK บอก RabbitMQ ว่าประมวลผลเสร็จแล้ว ลบออกจากคิวได้
				_ = d.Ack(false)
			}
		}
	}
}
