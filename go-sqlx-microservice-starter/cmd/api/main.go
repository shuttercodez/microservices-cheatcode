package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go-sqlx-microservice-starter/internal/config"
	"go-sqlx-microservice-starter/internal/order"
	"go-sqlx-microservice-starter/internal/platform/cache"
	"go-sqlx-microservice-starter/internal/platform/database"
	"go-sqlx-microservice-starter/internal/platform/queue"
	"go-sqlx-microservice-starter/internal/product"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Initialize structured JSON logger following microservice standards
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 0. Load configuration from YAML (with Cloud Vault / Environment Variable overrides)
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("Failed to load application configuration", "path", configPath, "error", err)
		os.Exit(1)
	}

	// 1. Connect to PostgreSQL using pgxpool (pure SQL, no ORM)
	db, err := database.NewPostgresDB(cfg.Database.DSN())
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 2. Connect to Redis Cache
	redisCache, err := cache.NewRedisCache(cfg.Redis.Addr)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisCache.Close()

	// 3. Connect to RabbitMQ and declare queues
	mq, err := queue.NewRabbitMQ(cfg.RabbitMQ.URL)
	if err != nil {
		slog.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer mq.Close()

	if err := mq.DeclareQueue(product.QueueStockChanged); err != nil {
		slog.Error("Failed to declare product queue", "error", err)
		os.Exit(1)
	}
	if err := mq.DeclareQueue(order.QueueOrderCreated); err != nil {
		slog.Error("Failed to declare order queue", "error", err)
		os.Exit(1)
	}

	// 4. Start background queue consumers
	ctx, cancelConsumer := context.WithCancel(context.Background())
	go product.StartQueueConsumer(ctx, mq.Conn, product.QueueStockChanged)
	go order.StartQueueConsumer(ctx, mq.Conn, order.QueueOrderCreated)

	// 5. Dependency Injection (Repository, Cache & Producer -> Service -> Handler)
	// --- Product Domain ---
	productRepo := product.NewRepository(db)
	productCache := product.NewCache(redisCache, cfg.Redis.TTL())
	productProducer := product.NewProducer(mq)
	productService := product.NewService(productRepo, productCache, productProducer)
	productHandler := product.NewHandler(productService)

	// --- Order Domain (Inject productService for cross-domain stock verification and deduction) ---
	orderRepo := order.NewRepository(db)
	orderCache := order.NewCache(redisCache, cfg.Redis.TTL())
	orderProducer := order.NewProducer(mq)
	orderService := order.NewService(orderRepo, orderCache, productService, orderProducer)
	orderHandler := order.NewHandler(orderService)

	// 6. Configure Fiber application
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		DisableStartupMessage: true,
	})

	// Register panic recovery and CORS middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// --- Microservice Health & Readiness Probes ---
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "alive"})
	})

	app.Get("/readyz", func(c *fiber.Ctx) error {
		if err := database.Ping(c.UserContext(), db); err != nil || redisCache.Ping(c.UserContext()) != nil || mq.IsClosed() {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not ready"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ready"})
	})

	// --- RESTful API Endpoints ---
	productHandler.RegisterRoutes(app)
	orderHandler.RegisterRoutes(app)

	// 7. Start Fiber server and configure graceful shutdown
	listenAddr := fmt.Sprintf(":%d", cfg.App.Port)
	go func() {
		slog.Info("Starting Fiber HTTP server", "app", cfg.App.Name, "env", cfg.App.Env, "addr", listenAddr)
		if err := app.Listen(listenAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "error", err)
		}
	}()

	// Wait for termination signals (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down service gracefully...")
	cancelConsumer() // Stop background consumers

	// Gracefully shut down Fiber server (draining active connections)
	if err := app.ShutdownWithTimeout(cfg.App.ShutdownTimeout()); err != nil {
		slog.Error("Fiber forced to shutdown", "error", err)
	}
	slog.Info("Service exited cleanly")
}
