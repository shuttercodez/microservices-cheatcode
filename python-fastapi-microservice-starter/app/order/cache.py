from __future__ import annotations

import logging
from abc import ABC, abstractmethod
from typing import Optional

from app.order.model import Order, PaginatedOrders
from app.platform.cache import RedisCache

logger = logging.getLogger(__name__)


class Cache(ABC):
    """Defines the distributed caching contract for orders."""

    @abstractmethod
    async def get(self, order_id: str) -> Optional[Order]:
        raise NotImplementedError

    @abstractmethod
    async def set(self, order: Order) -> None:
        raise NotImplementedError

    @abstractmethod
    async def get_list(self, page: int, limit: int) -> Optional[PaginatedOrders]:
        raise NotImplementedError

    @abstractmethod
    async def set_list(self, page: int, limit: int, list_data: PaginatedOrders) -> None:
        raise NotImplementedError

    @abstractmethod
    async def invalidate_lists(self) -> None:
        raise NotImplementedError


def _cache_key(order_id: str) -> str:
    return f"order:{order_id}"


def list_cache_key(page: int, limit: int) -> str:
    return f"orders:list:page:{page}:limit:{limit}"


class RedisOrderCache(Cache):
    """Redis-backed Order Cache with configurable TTL."""

    def __init__(self, redis: Optional[RedisCache], ttl_seconds: int) -> None:
        self._redis = redis
        self._ttl_seconds = ttl_seconds

    async def get(self, order_id: str) -> Optional[Order]:
        if self._redis is None:
            return None

        try:
            data = await self._redis.get(_cache_key(order_id))
            if data is None:
                return None
            order = Order.model_validate_json(data)
            logger.info("⚡ [Redis Cache HIT - Order]", extra={"id": order_id})
            return order
        except Exception as exc:
            logger.warning("Redis order cache get failed", extra={"id": order_id, "error": str(exc)})
            return None

    async def set(self, order: Order) -> None:
        if self._redis is None or order is None:
            return

        try:
            payload = order.model_dump_json().encode("utf-8")
            await self._redis.set(
                _cache_key(str(order.id)),
                payload,
                self._ttl_seconds,
            )
        except Exception as exc:
            logger.warning(
                "Redis order cache set failed",
                extra={"id": str(order.id), "error": str(exc)},
            )

    async def get_list(self, page: int, limit: int) -> Optional[PaginatedOrders]:
        if self._redis is None:
            return None

        key = list_cache_key(page, limit)
        try:
            data = await self._redis.get(key)
            if data is None:
                return None
            list_data = PaginatedOrders.model_validate_json(data)
            logger.info("⚡ [Redis Cache HIT - Order List]", extra={"page": page, "limit": limit})
            return list_data
        except Exception as exc:
            logger.warning("Redis order cache get list failed", extra={"key": key, "error": str(exc)})
            return None

    async def set_list(self, page: int, limit: int, list_data: PaginatedOrders) -> None:
        if self._redis is None or list_data is None:
            return

        key = list_cache_key(page, limit)
        try:
            payload = list_data.model_dump_json().encode("utf-8")
            await self._redis.set(key, payload, self._ttl_seconds)
        except Exception as exc:
            logger.warning("Redis order cache set list failed", extra={"key": key, "error": str(exc)})

    async def invalidate_lists(self) -> None:
        if self._redis is None:
            return

        try:
            await self._redis.delete_pattern("orders:list:*")
        except Exception as exc:
            logger.warning("Redis order cache list invalidation failed", extra={"error": str(exc)})
