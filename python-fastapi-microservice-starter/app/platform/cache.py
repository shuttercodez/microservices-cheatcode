from __future__ import annotations

from typing import Optional

import redis.asyncio as aioredis


class RedisCache:
    """Wraps redis.asyncio.Redis for non-blocking distributed caching operations."""

    def __init__(self, client: aioredis.Redis) -> None:
        self.client = client

    @classmethod
    async def connect(cls, addr: str) -> RedisCache:
        url = addr if addr.startswith(("redis://", "rediss://")) else f"redis://{addr}"
        client = aioredis.from_url(url, socket_connect_timeout=3.0)
        cache = cls(client)
        await cache.ping()
        return cache

    async def ping(self) -> None:
        await self.client.ping()

    async def get(self, key: str) -> Optional[bytes]:
        return await self.client.get(key)

    async def set(self, key: str, value: bytes, ttl_seconds: int) -> None:
        await self.client.set(key, value, ex=ttl_seconds)

    async def delete(self, key: str) -> None:
        await self.client.delete(key)

    async def delete_pattern(self, pattern: str) -> None:
        keys = await self.client.keys(pattern)
        if keys:
            await self.client.delete(*keys)

    async def close(self) -> None:
        await self.client.aclose()
