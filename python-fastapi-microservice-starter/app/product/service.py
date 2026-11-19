from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Optional

from app.product.cache import Cache
from app.product.model import (
    CreateProductRequest,
    ErrInvalidPayload,
    ErrNegativeStock,
    PaginatedProducts,
    PatchStockRequest,
    Product,
)
from app.product.producer import Producer
from app.product.repository import Repository


def _normalize_pagination(page: int, limit: int) -> tuple[int, int]:
    if page < 1:
        page = 1
    if limit < 1:
        limit = 10
    elif limit > 100:
        limit = 100
    return page, limit


def _calc_total_pages(total: int, limit: int) -> int:
    if total == 0 or limit <= 0:
        return 0
    return (total + limit - 1) // limit


class Service(ABC):
    """Defines the business logic contract for managing products."""

    @abstractmethod
    async def create(self, req: CreateProductRequest) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def get_all(self, page: int = 1, limit: int = 10) -> PaginatedProducts:
        raise NotImplementedError

    @abstractmethod
    async def get_by_id(self, product_id: str) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def update(self, product_id: str, req: CreateProductRequest) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def patch_stock(self, product_id: str, req: PatchStockRequest) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def delete(self, product_id: str) -> None:
        raise NotImplementedError


class ProductService(Service):
    """Product Service with injected Repository, Cache, and Producer dependencies."""

    def __init__(
        self,
        repo: Repository,
        cache: Optional[Cache],
        producer: Producer,
    ) -> None:
        self._repo = repo
        self._cache = cache
        self._producer = producer

    async def create(self, req: CreateProductRequest) -> Product:
        if not req.sku or not req.name or req.price < 0 or req.stock < 0:
            raise ErrInvalidPayload()

        product = Product(
            sku=req.sku,
            name=req.name,
            price=req.price,
            stock=req.stock,
        )

        created = await self._repo.create(product)

        if self._cache is not None:
            await self._cache.set(created)
            await self._cache.invalidate_lists()
        await self._producer.publish_stock_changed(created)
        return created

    async def get_all(self, page: int = 1, limit: int = 10) -> PaginatedProducts:
        page, limit = _normalize_pagination(page, limit)

        if self._cache is not None:
            cached = await self._cache.get_list(page, limit)
            if cached is not None:
                return cached

        products, total = await self._repo.find_all(page, limit)
        res = PaginatedProducts(
            data=products,
            page=page,
            limit=limit,
            total=total,
            total_pages=_calc_total_pages(total, limit),
        )

        if self._cache is not None:
            await self._cache.set_list(page, limit, res)
        return res

    async def get_by_id(self, product_id: str) -> Product:
        """Implements the Cache-Aside pattern: check Redis first, fallback to DB on miss, and populate Redis."""
        if self._cache is not None:
            cached = await self._cache.get(product_id)
            if cached is not None:
                return cached

        product = await self._repo.find_by_id(product_id)

        if self._cache is not None:
            await self._cache.set(product)
        return product

    async def update(self, product_id: str, req: CreateProductRequest) -> Product:
        product = await self._repo.find_by_id(product_id)

        if not req.sku or not req.name or req.price < 0 or req.stock < 0:
            raise ErrInvalidPayload()

        product.sku = req.sku
        product.name = req.name
        product.price = req.price
        product.stock = req.stock
        product.version += 1

        updated = await self._repo.update(product)

        if self._cache is not None:
            await self._cache.set(updated)
            await self._cache.invalidate_lists()
        await self._producer.publish_stock_changed(updated)
        return updated

    async def patch_stock(self, product_id: str, req: PatchStockRequest) -> Product:
        if req.stock < 0:
            raise ErrNegativeStock()

        # Always read current version directly from the database for optimistic locking
        product = await self._repo.find_by_id(product_id)
        updated = await self._repo.update_stock_with_version(product, req.stock)

        # Refresh Redis cache with the new stock and incremented version
        if self._cache is not None:
            await self._cache.set(updated)
            await self._cache.invalidate_lists()
        await self._producer.publish_stock_changed(updated)
        return updated

    async def delete(self, product_id: str) -> None:
        await self._repo.delete(product_id)
        if self._cache is not None:
            await self._cache.delete(product_id)
            await self._cache.invalidate_lists()
