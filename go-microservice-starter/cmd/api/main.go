package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-microservice-starter/internal/platform/database"
	"go-microservice-starter/internal/platform/queue"
	"go-microservice-starter/internal/product"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// ใช้ Structured JSON Logger ตามมาตรฐาน Microservices
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. เชื่อมต่อ PostgreSQL ด้วย GORM
	dsn := "host=localhost user=appuser password=apppassword dbname=inventory_db port=5432 sslmode=disable"
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}

	// 2. เชื่อมต่อ RabbitMQ และสร้าง Queue
	mq, err := queue.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	if err != nil {
		slog.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer mq.Close()

	if err := mq.DeclareQueue(product.QueueStockChanged); err != nil {
		slog.Error("Failed to declare queue", "error", err)
		os.Exit(1)
	}

	// 3. รัน Background Consumer
	ctx, cancelConsumer := context.WithCancel(context.Background())
	go product.StartQueueConsumer(ctx, mq.Conn, product.QueueStockChanged)

	// 4. Dependency Injection (Repository -> Service -> Handler)
	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo, mq)
	productHandler := product.NewHandler(productService)

	// 5. ตั้งค่า Fiber App
	app := fiber.New(fiber.Config{
		// ช่วยปิดการแสดงผลหน้า Banner ตอนเริ่มโปรแกรมของ Fiber เพื่อให้ Log อ่านง่ายขึ้น
		DisableStartupMessage: true,
	})

	// ใส่ Middleware กันโปรแกรมแครช (Recover)
	app.Use(recover.New())

	// --- Microservice Health & Readiness Probes ---
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "alive"})
	})

	app.Get("/readyz", func(c *fiber.Ctx) error {
		if err := database.Ping(db); err != nil || mq.IsClosed() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ready"})
	})

	// --- RESTful API Endpoints ---
	productHandler.RegisterRoutes(app)

	// 6. รัน Fiber Server และตั้งค่า Graceful Shutdown
	go func() {
		slog.Info("Starting Fiber HTTP server on :8080")
		if err := app.Listen(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "error", err)
		}
	}()

	// รอรับสัญญาณปิดระบบ
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down service gracefully...")
	cancelConsumer() // หยุดการทำงานของ Consumer ใน Background

	// สั่งปิดตัวแอป Fiber (ระบบจะเคลียร์ Connection ที่ค้างอยู่ให้อัตโนมัติ)
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		slog.Error("Fiber forced to shutdown", "error", err)
	}
	slog.Info("Service exited cleanly")
}
