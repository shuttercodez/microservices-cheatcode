from __future__ import annotations

import asyncio
import json
import logging
import os
import sys
from contextlib import asynccontextmanager
from datetime import datetime, timezone

import uvicorn
from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

import app.order as order
import app.product as product
from app.config import Config
from app.platform import PostgresDatabase, RabbitMQ, RedisCache


class JSONFormatter(logging.Formatter):
    """Formats logs as structured JSON matching Go slog / Rust tracing output."""

    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "time": datetime.now(timezone.utc).isoformat(),
            "level": record.levelname,
            "msg": record.getMessage(),
        }
        for key in ("id", "sku", "stock", "new_stock", "event_id", "order_id", "product_id", "quantity", "total_price", "error", "path", "app", "env", "addr"):
            if hasattr(record, key):
                payload[key] = getattr(record, key)
        return json.dumps(payload, ensure_ascii=False)


def configure_logging() -> None:
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(JSONFormatter())
    root = logging.getLogger()
    root.handlers = [handler]
    root.setLevel(logging.INFO)


logger = logging.getLogger(__name__)


def create_app() -> FastAPI:
    configure_logging()

    config_path = os.getenv("CONFIG_PATH") or "config/config.yaml"
    cfg = Config.load(config_path)

    state: dict = {}

    @asynccontextmanager
    async def lifespan(app_instance: FastAPI):
        # 1. Connect to PostgreSQL using SQLAlchemy 2.0 Async + asyncpg
        db = await PostgresDatabase.connect(cfg.database.dsn())

        # 2. Connect to Redis Cache
        redis_cache = await RedisCache.connect(cfg.redis.addr)

        # 3. Connect to RabbitMQ and declare queues
        mq = await RabbitMQ.connect(cfg.rabbitmq.url)
        await mq.declare_queue(product.QUEUE_STOCK_CHANGED)
        await mq.declare_queue(order.QUEUE_ORDER_CREATED)

        # 4. Start background queue consumers
        stop_event = asyncio.Event()
        prod_consumer_task = asyncio.create_task(
            product.start_queue_consumer(stop_event, mq.conn, product.QUEUE_STOCK_CHANGED)
        )
        ord_consumer_task = asyncio.create_task(
            order.start_queue_consumer(stop_event, mq.conn, order.QUEUE_ORDER_CREATED)
        )

        # 5. Dependency Injection (Repository, Cache & Producer -> Service -> Handler)
        # --- Product Domain ---
        product_repo = product.SQLAlchemyRepository(db.session_factory)
        product_cache = product.RedisProductCache(redis_cache)
        product_producer = product.RabbitMQProducer(mq)
        product_service = product.ProductService(product_repo, product_cache, product_producer)
        product_handler = product.Handler(product_service)

        # --- Order Domain (Inject product_service for cross-domain stock verification and deduction) ---
        order_repo = order.SQLAlchemyRepository(db.session_factory)
        order_producer = order.RabbitMQProducer(mq)
        order_service = order.OrderService(order_repo, product_service, order_producer)
        order_handler = order.Handler(order_service)

        app_instance.include_router(product_handler.router)
        app_instance.include_router(order_handler.router)

        state["db"] = db
        state["redis"] = redis_cache
        state["mq"] = mq

        logger.info(
            "Starting FastAPI HTTP server",
            extra={"app": cfg.app.name, "env": cfg.app.env, "addr": f":{cfg.app.port}"},
        )
        try:
            yield
        finally:
            logger.info("Shutting down service gracefully...")
            stop_event.set()
            prod_consumer_task.cancel()
            ord_consumer_task.cancel()
            await asyncio.gather(prod_consumer_task, ord_consumer_task, return_exceptions=True)
            await mq.close()
            await redis_cache.close()
            await db.close()
            logger.info("Service exited cleanly")

    app_instance = FastAPI(title=cfg.app.name, lifespan=lifespan)

    @app_instance.exception_handler(RequestValidationError)
    async def validation_exception_handler(_request: Request, _exc: RequestValidationError):
        return JSONResponse(
            status_code=status.HTTP_400_BAD_REQUEST,
            content={"error": "cannot parse JSON"},
        )

    @app_instance.get("/healthz")
    async def healthz():
        return {"status": "alive"}

    @app_instance.get("/readyz")
    async def readyz():
        db: PostgresDatabase | None = state.get("db")
        redis_cache: RedisCache | None = state.get("redis")
        mq: RabbitMQ | None = state.get("mq")
        if db is None or redis_cache is None or mq is None:
            return JSONResponse(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                content={"status": "not ready"},
            )
        try:
            await db.ping()
            await redis_cache.ping()
            if mq.is_closed():
                raise RuntimeError("RabbitMQ closed")
            return {"status": "ready"}
        except Exception:
            return JSONResponse(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                content={"status": "not ready"},
            )

    return app_instance


app = create_app()

if __name__ == "__main__":
    config_path = os.getenv("CONFIG_PATH") or "config/config.yaml"
    cfg = Config.load(config_path)
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=cfg.app.port,
        timeout_graceful_shutdown=cfg.app.shutdown_timeout,
    )
