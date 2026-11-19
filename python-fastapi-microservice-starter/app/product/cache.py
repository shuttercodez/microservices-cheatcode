from __future__ import annotations

import logging
from abc import ABC, abstractmethod
from typing import Optional

from app.platform.cache import RedisCache
from app.product.model import PaginatedProducts, Product

logger = logging.getLogger(__name__)


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

    @abstractmethod
    async def get_list(self, page: int, limit: int) -> Optional[PaginatedProducts]:
        raise NotImplementedError

    @abstractmethod
    async def set_list(self, page: int, limit: int, list_data: PaginatedProducts) -> None:
        raise NotImplementedError

    @abstractmethod
    async def invalidate_lists(self) -> None:
        raise NotImplementedError


def _cache_key(product_id: str) -> str:
    return f"product:{product_id}"


def list_cache_key(page: int, limit: int) -> str:
    return f"products:list:page:{page}:limit:{limit}"


class RedisProductCache(Cache):
    """Redis-backed Product Cache with configurable TTL."""

    def __init__(self, redis: Optional[RedisCache], ttl_seconds: int) -> None:
        self._redis = redis
        self._ttl_seconds = ttl_seconds

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
                self._ttl_seconds,
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

    async def get_list(self, page: int, limit: int) -> Optional[PaginatedProducts]:
        if self._redis is None:
            return None

        key = list_cache_key(page, limit)
        try:
            data = await self._redis.get(key)
            if data is None:
                return None
            list_data = PaginatedProducts.model_validate_json(data)
            logger.info("⚡ [Redis Cache HIT - Product List]", extra={"page": page, "limit": limit})
            return list_data
        except Exception as exc:
            logger.warning("Redis cache get list failed", extra={"key": key, "error": str(exc)})
            return None

    async def set_list(self, page: int, limit: int, list_data: PaginatedProducts) -> None:
        if self._redis is None or list_data is None:
            return

        key = list_cache_key(page, limit)
        try:
            payload = list_data.model_dump_json().encode("utf-8")
            await self._redis.set(key, payload, self._ttl_seconds)
        except Exception as exc:
            logger.warning("Redis cache set list failed", extra={"key": key, "error": str(exc)})

    async def invalidate_lists(self) -> None:
        if self._redis is None:
            return

        try:
            await self._redis.delete_pattern("products:list:*")
        except Exception as exc:
            logger.warning("Redis cache list invalidation failed", extra={"error": str(exc)})
