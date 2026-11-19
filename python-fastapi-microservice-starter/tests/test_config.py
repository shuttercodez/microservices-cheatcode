from __future__ import annotations

from pathlib import Path

import pytest

from app.config import Config


def test_load_config_yaml_and_vault_env_override(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    yaml_path = tmp_path / "config.yaml"
    yaml_content = """
app:
  name: "test-service"
  env: "staging"
  port: 9090
  shutdown_timeout_seconds: 10
database:
  host: "db.internal"
  port: 5432
  user: "yaml_user"
  password: "yaml_password"
  name: "orders_db"
  ssl_mode: "require"
redis:
  addr: "redis.internal:6379"
  ttl_seconds: 600
rabbitmq:
  url: "amqp://yaml:yaml@mq.internal:5672/"
"""
    yaml_path.write_text(yaml_content, encoding="utf-8")

    # 1. Loads values from config.yaml accurately
    cfg = Config.load(yaml_path)
    assert cfg.app.port == 9090
    assert cfg.database.user == "yaml_user"

    # 2. Cloud Vault environment variables override YAML secrets
    monkeypatch.setenv("DB_PASSWORD", "vault_super_secret_pw")
    monkeypatch.setenv("REDIS_ADDR", "prod-redis.cache.windows.net:6380")

    cfg_overridden = Config.load(yaml_path)
    assert cfg_overridden.database.password == "vault_super_secret_pw"
    assert cfg_overridden.redis.addr == "prod-redis.cache.windows.net:6380"
