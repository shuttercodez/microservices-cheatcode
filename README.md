# Microservices Cheatcode (Go, Rust & Python Starter Templates)

A collection of **Production-Ready Microservice (Inventory & Order Service)** starter templates built with the exact same architecture (**Clean / Layered Architecture + Dependency Injection**) across **3 languages and 4 community-recommended stacks**.

Designed for side-by-side comparison of project structure, tech stacks, database access patterns (Full ORM vs. SQL Toolkit / Micro-ORM), distributed caching (Redis with Cache-Aside & list invalidation), pagination, and event-driven messaging (RabbitMQ).

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
   * Clean separation of concerns across **`Handler` (HTTP Controller)** -> **`Service` (Business Logic)** -> **`Repository` (Database)** / **`Cache` (Redis)** / **`Producer` (RabbitMQ)** in both the `product` and `order` domains using interfaces, traits, or abstract base classes for fast, isolated unit testing with in-memory mocks.
2. **Cross-Domain Orchestration (`Order` -> `Product`)**
   * `OrderService` depends on `ProductService` via interface injection to verify product availability and deduct stock atomically before persisting orders.
3. **Full Redis Caching (Products & Orders + Paginated Lists)**
   * Single-item endpoints (`GET /api/v1/products/:id`, `GET /api/v1/orders/:id`) and paginated list endpoints (`GET /api/v1/products?page=1&limit=10`, `GET /api/v1/orders?page=1&limit=10`) use the **Cache-Aside** pattern with a configurable TTL (`redis.ttl_seconds` in `config/config.yaml`, overridable via `REDIS_TTL_SECONDS`).
   * Write operations (`POST`, `PUT`, `PATCH`, `DELETE`) immediately update or delete individual entity cache keys and invalidate cached list pages (`products:list:*`, `orders:list:*`) to prevent stale reads.
4. **Standardized Pagination (`?page=1&limit=10`)**
   * Both `GET /api/v1/products` and `GET /api/v1/orders` support `page` (default `1`) and `limit` (default `10`, max `100`) query parameters, returning total counts and calculated `total_pages`.
5. **Optimistic Locking (Concurrency Control)**
   * The `products` table uses a `version` column. Stock updates (`PATCH /api/v1/products/:id/stock`) execute `WHERE id = $1 AND version = $2` while incrementing `version + 1`. If a concurrent request modifies the record first (`RowsAffected == 0`), the service returns `409 Conflict`.
6. **Soft Deletes**
   * `DELETE /api/v1/products/:id` sets `deleted_at = NOW()` instead of physically removing rows, and all read/update queries filter by `WHERE deleted_at IS NULL`.
7. **Event-Driven Messaging (RabbitMQ Producer & Background Consumer)**
   * Publishes persistent JSON domain events (`delivery_mode = 2`) to durable queues (`inventory.stock_changed` and `order.created`) and runs background worker tasks to consume and acknowledge messages.
8. **Single-Source Configuration, Cloud Vault Overrides & Graceful Shutdown**
   * `config/config.yaml` serves as the single source of truth for default settings (no duplicate hardcoded defaults in code), with runtime environment variable overrides for Cloud Secret Managers / Vaults.
   * Exposes Kubernetes-ready `/healthz` (Liveness) and `/readyz` (Readiness checking PostgreSQL, Redis, and RabbitMQ connectivity) probes.
   * Handles `SIGINT` / `SIGTERM` signals to stop background consumers and drain active HTTP connections cleanly.

---

## ⚡ Redis Cache Key Strategy & Invalidation

| Domain | Redis Key Pattern | Stored Payload | Populated On | Invalidated / Updated On |
| :--- | :--- | :--- | :--- | :--- |
| **Product (Single)** | `product:<uuid>` | Single `Product` JSON | `POST`, `GET /:id`, `PUT /:id`, `PATCH /:id/stock` | Updated on `PUT` / `PATCH`; deleted on `DELETE /:id` |
| **Product (List)** | `products:list:page:<p>:limit:<l>` | `PaginatedProducts` JSON | `GET /api/v1/products?page=<p>&limit=<l>` | Cleared (`products:list:*`) on any product `POST`, `PUT`, `PATCH`, or `DELETE` |
| **Order (Single)** | `order:<uuid>` | Single `Order` JSON | `POST /api/v1/orders`, `GET /api/v1/orders/:id` | Populated on order creation and single-order reads |
| **Order (List)** | `orders:list:page:<p>:limit:<l>` | `PaginatedOrders` JSON | `GET /api/v1/orders?page=<p>&limit=<l>` | Cleared (`orders:list:*`) whenever a new order is created (`POST /api/v1/orders`) |

---

## ⚙️ Configuration & Environment Variable Overrides

All 4 projects load `config/config.yaml` (or the path specified by `CONFIG_PATH`):

```yaml
app:
  name: "inventory-order-service"
  env: "development"
  port: 8080
  shutdown_timeout_seconds: 5

database:
  host: "localhost"
  port: 5432
  user: "appuser"
  password: "apppassword"
  name: "inventory_db"
  ssl_mode: "disable"

redis:
  addr: "localhost:6379"
  ttl_seconds: 300

rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
```

Any value in `config.yaml` can be overridden at runtime via Environment Variables (e.g., injected from Kubernetes Secrets, AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault):

| Environment Variable | Overrides YAML Key | Description |
| :--- | :--- | :--- |
| `CONFIG_PATH` | *(File path)* | Custom path to `config.yaml` (defaults to `config/config.yaml`) |
| `APP_ENV` | `app.env` | Runtime environment (`development`, `staging`, `production`) |
| `APP_PORT` | `app.port` | HTTP server listening port |
| `DB_DSN` | `database.*` | Full PostgreSQL connection string / URI (takes highest precedence) |
| `DB_HOST` | `database.host` | PostgreSQL hostname |
| `DB_PORT` | `database.port` | PostgreSQL port |
| `DB_USER` | `database.user` | PostgreSQL username |
| `DB_PASSWORD` | `database.password` | PostgreSQL password |
| `DB_NAME` | `database.name` | PostgreSQL database name |
| `DB_SSLMODE` | `database.ssl_mode` | PostgreSQL SSL mode (`disable`, `require`, etc.) |
| `REDIS_ADDR` | `redis.addr` | Redis server address (`host:port` or `redis://...`) |
| `REDIS_TTL_SECONDS` | `redis.ttl_seconds` | Redis cache TTL in seconds |
| `RABBITMQ_URL` | `rabbitmq.url` | RabbitMQ AMQP connection URL (`amqp://...`) |

---

## 🔌 RESTful API Endpoints (Identical Across All Projects)

| Method | Endpoint | Description | Status Codes |
| :--- | :--- | :--- | :--- |
| `GET` | `/healthz` | Liveness probe | `200 OK` |
| `GET` | `/readyz` | Readiness probe (verifies PostgreSQL, Redis, and RabbitMQ connectivity) | `200 OK`, `503 Service Unavailable` |
| `POST` | `/api/v1/products` | Create a new product, cache it in Redis, invalidate product lists, and publish a stock event | `201 Created`, `400 Bad Request`, `409 Conflict` |
| `GET` | `/api/v1/products` | List active products with pagination (`?page=1&limit=10`) and Redis list caching | `200 OK` |
| `GET` | `/api/v1/products/:id` | Get a product by ID using the **Cache-Aside** pattern | `200 OK`, `404 Not Found` |
| `PUT` | `/api/v1/products/:id` | Replace product details, refresh Redis cache, invalidate product lists, and publish an event | `200 OK`, `400 Bad Request`, `404 Not Found` |
| `PATCH` | `/api/v1/products/:id/stock` | Update product stock using **Optimistic Locking (`version`)**, refresh cache, and publish an event | `200 OK`, `400 Bad Request`, `404 Not Found`, `409 Conflict` |
| `DELETE` | `/api/v1/products/:id` | **Soft-delete** a product and invalidate its Redis cache entry & product list caches | `204 No Content`, `404 Not Found` |
| `POST` | `/api/v1/orders` | Create an order, verify & deduct product stock, cache order, invalidate order lists, and publish event | `201 Created`, `400 Bad Request`, `404 Not Found`, `409 Conflict` |
| `GET` | `/api/v1/orders` | List active orders with pagination (`?page=1&limit=10`) ordered by `created_at DESC` with Redis caching | `200 OK` |
| `GET` | `/api/v1/orders/:id` | Get a single order by ID using the **Cache-Aside** pattern | `200 OK`, `404 Not Found` |

### Paginated Response Schema (`GET /api/v1/products` & `GET /api/v1/orders`)
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "sku": "IPHONE-16",
      "name": "iPhone 16 Pro",
      "price": 39900,
      "stock": 50,
      "version": 1,
      "created_at": "2026-11-19T06:00:00Z",
      "updated_at": "2026-11-19T06:00:00Z"
    }
  ],
  "page": 1,
  "limit": 10,
  "total": 1,
  "total_pages": 1
}
```

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
