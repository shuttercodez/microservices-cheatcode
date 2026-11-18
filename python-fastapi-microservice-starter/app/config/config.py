from __future__ import annotations

import os
from pathlib import Path
from typing import Union

import yaml
from pydantic import BaseModel, Field


class AppConfig(BaseModel):
    name: str = "inventory-order-service"
    env: str = "development"
    port: int = 8080
    shutdown_timeout_seconds: int = 5

    @property
    def shutdown_timeout(self) -> int:
        return self.shutdown_timeout_seconds if self.shutdown_timeout_seconds > 0 else 5


class DatabaseConfig(BaseModel):
    host: str = "localhost"
    port: int = 5432
    user: str = "appuser"
    password: str = "apppassword"
    name: str = "inventory_db"
    ssl_mode: str = "disable"

    def dsn(self) -> str:
        """Builds the PostgreSQL connection URI for SQLAlchemy + asyncpg.
        If DB_DSN is explicitly provided by a Cloud Vault, it takes highest precedence.
        """
        env_dsn = os.getenv("DB_DSN")
        if env_dsn:
            return env_dsn
        return f"postgresql+asyncpg://{self.user}:{self.password}@{self.host}:{self.port}/{self.name}"


class RedisConfig(BaseModel):
    addr: str = "localhost:6379"


class RabbitMQConfig(BaseModel):
    url: str = "amqp://guest:guest@localhost:5672/"


class Config(BaseModel):
    app: AppConfig = Field(default_factory=AppConfig)
    database: DatabaseConfig = Field(default_factory=DatabaseConfig)
    redis: RedisConfig = Field(default_factory=RedisConfig)
    rabbitmq: RabbitMQConfig = Field(default_factory=RabbitMQConfig)

    @classmethod
    def load(cls, path: Union[str, Path]) -> Config:
        """Reads the YAML configuration file from the given path and applies environment variable
        overrides (used when deploying with AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault).
        """
        data = Path(path).read_text(encoding="utf-8")
        raw = yaml.safe_load(data) or {}
        cfg = cls.model_validate(raw)
        cfg.apply_env_overrides()
        return cfg

    def apply_env_overrides(self) -> None:
        if v := os.getenv("APP_ENV"):
            self.app.env = v
        if v := os.getenv("APP_PORT"):
            try:
                self.app.port = int(v)
            except ValueError:
                pass

        if v := os.getenv("DB_HOST"):
            self.database.host = v
        if v := os.getenv("DB_PORT"):
            try:
                self.database.port = int(v)
            except ValueError:
                pass
        if v := os.getenv("DB_USER"):
            self.database.user = v
        if v := os.getenv("DB_PASSWORD"):
            self.database.password = v
        if v := os.getenv("DB_NAME"):
            self.database.name = v
        if v := os.getenv("DB_SSLMODE"):
            self.database.ssl_mode = v

        if v := os.getenv("REDIS_ADDR"):
            self.redis.addr = v
        if v := os.getenv("RABBITMQ_URL"):
            self.rabbitmq.url = v
