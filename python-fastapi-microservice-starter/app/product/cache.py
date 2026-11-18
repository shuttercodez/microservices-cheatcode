from __future__ import annotations

import logging
from abc import ABC, abstractmethod
from typing import Optional

from app.platform.cache import RedisCache
from app.product.model import Product

logger = logging.getLogger(__name__)

PRODUCT_CACHE_TTL_SECONDS = 5 * 60


class Cache(ABC):
    """Defines the distributed caching contract for products."""

    @abstractmethod
    async def get(self, product_id: str) -> Optional[Product]:
        raise NotImplementedError

    @abstractmethod
    async def set(self, product: Product) -> None:
        raise NotImplementedError

    @abstractmethod
    async def delete(self, product_id: str) -> None:
        raise NotImplementedError


def _cache_key(product_id: str) -> str:
    return f"product:{product_id}"


class RedisProductCache(Cache):
    """Redis-backed Product Cache."""

    def __init__(self, redis: Optional[RedisCache]) -> None:
        self._redis = redis

    async def get(self, product_id: str) -> Optional[Product]:
        if self._redis is None:
            return None

        try:
            data = await self._redis.get(_cache_key(product_id))
            if data is None:
                return None
            product = Product.model_validate_json(data)
            logger.info("⚡ [Redis Cache HIT]", extra={"id": product_id, "sku": product.sku})
            return product
        except Exception as exc:
            logger.warning("Redis cache get failed", extra={"id": product_id, "error": str(exc)})
            return None

    async def set(self, product: Product) -> None:
        if self._redis is None or product is None:
            return

        try:
            payload = product.model_dump_json().encode("utf-8")
            await self._redis.set(
                _cache_key(str(product.id)),
                payload,
                PRODUCT_CACHE_TTL_SECONDS,
            )
        except Exception as exc:
            logger.warning(
                "Redis cache set failed",
                extra={"id": str(product.id), "error": str(exc)},
            )

    async def delete(self, product_id: str) -> None:
        if self._redis is None:
            return

        try:
            await self._redis.delete(_cache_key(product_id))
        except Exception as exc:
            logger.warning(
                "Redis cache delete failed",
                extra={"id": product_id, "error": str(exc)},
            )
