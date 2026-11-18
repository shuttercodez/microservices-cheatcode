# Microservices Cheatcode (Go, Rust & Python Starter Templates)

A collection of **Production-Ready Microservice (Inventory & Order Service)** starter templates built with the exact same architecture (**Clean / Layered Architecture + Dependency Injection**) across **3 languages and 4 community-recommended stacks**.

Designed for side-by-side comparison of project structure, tech stacks, database access patterns (Full ORM vs. SQL Toolkit / Micro-ORM), distributed caching (Redis), and event-driven messaging (RabbitMQ).

---

## 📁 Projects in This Repository

1. **[`go-microservice-starter`](./go-microservice-starter)** — **Go + Fiber + GORM**
   * Ideal for teams coming from Full ORM backgrounds (e.g., .NET EF Core or Java Spring Data JPA/Hibernate) looking for rapid CRUD development.
2. **[`go-sqlx-microservice-starter`](./go-sqlx-microservice-starter)** — **Go + Fiber + `sqlx` (`pgx/v5`)**
   * Ideal for idiomatic Go teams seeking 100% explicit SQL control (similar to Dapper in .NET) and maximum database performance without ORM overhead.
3. **[`rust-microservice-starter`](./rust-microservice-starter)** — **Rust + Axum + `SQLx`**
   * Ideal for high-performance, low-latency services leveraging zero-cost abstractions, memory safety, and the standard Tokio async ecosystem.
4. **[`python-fastapi-microservice-starter`](./python-fastapi-microservice-starter)** — **Python + FastAPI + SQLAlchemy 2.0 (Async)**
   * Ideal for modern Python microservices built on full non-blocking Async I/O across the web layer, PostgreSQL, Redis, and RabbitMQ, powered by Pydantic v2 (Rust core).

---

## 📊 Tech Stack Comparison

| Layer / Component | 1. Go + ORM (`go-microservice-starter`) | 2. Go + SQLx (`go-sqlx-microservice-starter`) | 3. Rust (`rust-microservice-starter`) | 4. Python (`python-fastapi-microservice-starter`) |
| :--- | :--- | :--- | :--- | :--- |
| **Language & Runtime** | Go 1.27+ | Go 1.27+ | Rust (2021 Edition) + **Tokio** | Python 3.9+ (`asyncio` + **`uvloop`**) |
| **HTTP Web Framework** | **Fiber v2** (`gofiber/fiber/v2`) | **Fiber v2** (`gofiber/fiber/v2`) | **Axum 0.8** + **Tower HTTP** | **FastAPI** + **Uvicorn** |
| **Database Access / ORM** | **GORM** (`gorm.io/gorm`) | **`jmoiron/sqlx`** + **`pgx/v5`** | **`SQLx 0.9`** (`PgPool`, `FromRow`) | **SQLAlchemy 2.0 (`AsyncSession`)** + **`asyncpg`** |
| **Database Paradigm** | Full ORM (Active Record / Struct Tags) | Micro-ORM / SQL Mapper (Dapper-style) | Async SQL Toolkit + Struct Mapping | Modern Async ORM (`Mapped`, `select()`) |
| **DTO & Serialization** | `encoding/json` + Go Struct Tags | `encoding/json` + `db`/`json` Tags | **Serde** (`serde`, `serde_json`) | **Pydantic v2** (`BaseModel`) |
| **Config & Vault Override** | `gopkg.in/yaml.v3` + `os.Getenv` | `gopkg.in/yaml.v3` + `os.Getenv` | `serde_yaml_ng` + `std::env` | `PyYAML` + **Pydantic v2** + `os.getenv` |
| **Redis Client (Cache)** | **`go-redis/v9`** | **`go-redis/v9`** | **`redis`** (`aio::ConnectionManager`) | **`redis.asyncio`** |
| **RabbitMQ Client (AMQP)** | **`amqp091-go`** | **`amqp091-go`** | **`lapin`** | **`aio-pika`** |
| **Structured Logging** | `log/slog` (`JSONHandler`) | `log/slog` (`JSONHandler`) | **`tracing`** + **`tracing-subscriber` (JSON)** | `logging` + Custom `JSONFormatter` |
| **Dependency Injection** | Go `interface` | Go `interface` | `#[async_trait]` + `Arc<dyn Trait>` | Python `abc.ABC` (`@abstractmethod`) |
| **Unit & HTTP Testing** | `testing` + `net/http/httptest` (`app.Test`) | `testing` + `net/http/httptest` (`app.Test`) | `#[tokio::test]` + `tower::ServiceExt::oneshot` | **`pytest`** + **`pytest-asyncio`** + **`httpx`** |

---

## 🏗️ Architecture & Core Microservice Features

Every project in this repository implements the exact same production-grade microservice patterns:

1. **Layered Architecture & Explicit Dependency Injection**
   * Clean separation of concerns across **`Handler` (HTTP Controller)** -> **`Service` (Business Logic)** -> **`Repository` (Database)** / **`Cache` (Redis)** / **`Producer` (RabbitMQ)** using interfaces, traits, or abstract base classes for fast, isolated unit testing with in-memory mocks.
2. **Cross-Domain Orchestration (`Order` -> `Product`)**
   * `OrderService` depends on `ProductService` via interface injection to verify product availability and deduct stock atomically before persisting orders.
3. **Cache-Aside Pattern with Redis**
   * `GET /api/v1/products/:id` checks Redis first (`product:<id>`, 5-minute TTL). On a cache miss, it queries PostgreSQL and populates Redis. Updates and deletes immediately refresh or invalidate the cache entry.
4. **Optimistic Locking (Concurrency Control)**
   * The `products` table uses a `version` column. Stock updates (`PATCH /api/v1/products/:id/stock`) execute `WHERE id = $1 AND version = $2` while incrementing `version + 1`. If a concurrent request modifies the record first (`RowsAffected == 0`), the service returns `409 Conflict`.
5. **Soft Deletes**
   * `DELETE /api/v1/products/:id` sets `deleted_at = NOW()` instead of physically removing rows, and all read/update queries filter by `WHERE deleted_at IS NULL`.
6. **Event-Driven Messaging (RabbitMQ Producer & Background Consumer)**
   * Publishes persistent JSON domain events (`delivery_mode = 2`) to durable queues (`inventory.stock_changed` and `order.created`) and runs background worker tasks to consume and acknowledge messages.
7. **Cloud-Native Configuration, Health Probes & Graceful Shutdown**
   * Loads `config/config.yaml` with runtime environment variable overrides for Cloud Secret Managers / Vaults (`DB_DSN`, `DB_PASSWORD`, `REDIS_ADDR`, `RABBITMQ_URL`).
   * Exposes Kubernetes-ready `/healthz` (Liveness) and `/readyz` (Readiness checking PostgreSQL, Redis, and RabbitMQ connectivity) probes.
   * Handles `SIGINT` / `SIGTERM` signals to stop background consumers and drain active HTTP connections cleanly.

---

## 🔌 RESTful API Endpoints (Identical Across All Projects)

| Method | Endpoint | Description | Status Codes |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | Liveness probe | `200 OK` |
| `GET` | `/readyz` | Readiness probe (verifies PostgreSQL, Redis, and RabbitMQ connectivity) | `200 OK`, `503 Service Unavailable` |
| `POST` | `/api/v1/products` | Create a new product, cache it in Redis, and publish a stock event | `201 Created`, `400 Bad Request`, `409 Conflict` |
| `GET` | `/api/v1/products` | List all active products (`deleted_at IS NULL`) | `200 OK` |
| `GET` | `/api/v1/products/:id` | Get a product by ID using the **Cache-Aside** pattern | `200 OK`, `404 Not Found` |
| `PUT` | `/api/v1/products/:id` | Replace product details, refresh Redis cache, and publish an event | `200 OK`, `400 Bad Request`, `404 Not Found` |
| `PATCH` | `/api/v1/products/:id/stock` | Update product stock using **Optimistic Locking (`version`)** | `200 OK`, `400 Bad Request`, `404 Not Found`, `409 Conflict` |
| `DELETE` | `/api/v1/products/:id` | **Soft-delete** a product and invalidate its Redis cache entry | `204 No Content`, `404 Not Found` |
| `POST` | `/api/v1/orders` | Create an order, verify & deduct product stock, and publish an order event | `201 Created`, `400 Bad Request`, `404 Not Found`, `409 Conflict` |
| `GET` | `/api/v1/orders` | List all active orders ordered by `created_at DESC` | `200 OK` |
| `GET` | `/api/v1/orders/:id` | Get a single order by ID | `200 OK`, `404 Not Found` |

---

## 🚀 Getting Started & Running Tests

### 0. Start Infrastructure (PostgreSQL, Redis, RabbitMQ & Migrations)
You can start the shared backing services from any project folder using Docker Compose:
```bash
cd go-microservice-starter
docker compose up -d
```

---

### 1. `go-microservice-starter` (Go + Fiber + GORM)
```bash
cd go-microservice-starter

# Run unit & HTTP handler tests
go test -v ./...

# Start the HTTP server (port 8080)
go run ./cmd/api
```

---

### 2. `go-sqlx-microservice-starter` (Go + Fiber + `jmoiron/sqlx`)
```bash
cd go-sqlx-microservice-starter

# Run unit & HTTP handler tests
go test -v ./...

# Start the HTTP server (port 8080)
go run ./cmd/api
```

---

### 3. `rust-microservice-starter` (Rust + Axum + `SQLx`)
```bash
cd rust-microservice-starter

# Run unit & HTTP handler tests
cargo test

# Start the HTTP server (port 8080)
cargo run
```

---

### 4. `python-fastapi-microservice-starter` (Python + FastAPI + SQLAlchemy 2.0 Async)
```bash
cd python-fastapi-microservice-starter

# Create virtual environment & install dependencies (if not already set up)
python3 -m venv .venv
./.venv/bin/pip install -r requirements.txt

# Run unit & HTTP handler tests with Pytest
./.venv/bin/pytest -v

# Start the HTTP server (port 8080, Swagger UI available at http://localhost:8080/docs)
./.venv/bin/python -m app.main
```
