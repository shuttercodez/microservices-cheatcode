from __future__ import annotations

import uuid
from typing import Optional

import pytest

from app.product import (
    Cache,
    CreateProductRequest,
    ErrInvalidPayload,
    ErrNegativeStock,
    ErrProductNotFound,
    ErrSKUConflict,
    ErrVersionConflict,
    PatchStockRequest,
    Producer,
    Product,
    ProductService,
    Repository,
)


# ==================== 1. Mock Objects ====================
class MockRepository(Repository):
    def __init__(self) -> None:
        self.products: dict[str, Product] = {}
        self.find_by_id_calls: int = 0
        self.force_create_err: Optional[Exception] = None
        self.force_conflict_err: bool = False

    async def create(self, product: Product) -> Product:
        if self.force_create_err is not None:
            raise self.force_create_err
        copied = product.model_copy(deep=True)
        if not copied.id:
            copied.id = uuid.uuid4()
        copied.version = 1
        self.products[str(copied.id)] = copied
        return copied.model_copy(deep=True)

    async def find_all(self) -> list[Product]:
        return [p.model_copy(deep=True) for p in self.products.values()]

    async def find_by_id(self, product_id: str) -> Product:
        self.find_by_id_calls += 1
        p = self.products.get(product_id)
        if p is None:
            raise ErrProductNotFound()
        return p.model_copy(deep=True)

    async def update(self, product: Product) -> Product:
        copied = product.model_copy(deep=True)
        self.products[str(copied.id)] = copied
        return copied.model_copy(deep=True)

    async def update_stock_with_version(self, product: Product, new_stock: int) -> Product:
        if self.force_conflict_err:
            raise ErrVersionConflict()
        stored = self.products.get(str(product.id))
        if stored is None:
            raise ErrProductNotFound()
        if stored.version != product.version:
            raise ErrVersionConflict()
        stored.stock = new_stock
        stored.version += 1
        return stored.model_copy(deep=True)

    async def delete(self, product_id: str) -> None:
        if product_id not in self.products:
            raise ErrProductNotFound()
        del self.products[product_id]


class MockCache(Cache):
    def __init__(self) -> None:
        self.store: dict[str, Product] = {}

    async def get(self, product_id: str) -> Optional[Product]:
        p = self.store.get(product_id)
        return p.model_copy(deep=True) if p is not None else None

    async def set(self, product: Product) -> None:
        self.store[str(product.id)] = product.model_copy(deep=True)

    async def delete(self, product_id: str) -> None:
        self.store.pop(product_id, None)


class MockProducer(Producer):
    def __init__(self) -> None:
        self.published_events: list[Product] = []

    async def publish_stock_changed(self, product: Product) -> None:
        self.published_events.append(product.model_copy(deep=True))


# ==================== 2. Unit Tests for Product Service ====================
@pytest.mark.asyncio
async def test_product_service_create_success() -> None:
    repo = MockRepository()
    cache = MockCache()
    producer = MockProducer()
    svc = ProductService(repo, cache, producer)

    req = CreateProductRequest(sku="LAPTOP-001", name="Gaming Laptop", price=35000, stock=10)
    res = await svc.create(req)

    assert res.sku == "LAPTOP-001"
    assert len(producer.published_events) == 1


@pytest.mark.asyncio
async def test_product_service_create_negative_price_fails() -> None:
    repo = MockRepository()
    cache = MockCache()
    producer = MockProducer()
    svc = ProductService(repo, cache, producer)

    req = CreateProductRequest(sku="LAPTOP-002", name="Broken Laptop", price=-100, stock=5)
    with pytest.raises(ErrInvalidPayload):
        await svc.create(req)

    assert len(producer.published_events) == 0


@pytest.mark.asyncio
async def test_product_service_create_duplicate_sku_fails() -> None:
    repo = MockRepository()
    repo.force_create_err = ErrSKUConflict()
    cache = MockCache()
    producer = MockProducer()
    svc = ProductService(repo, cache, producer)

    req = CreateProductRequest(sku="DUP-SKU", name="Duplicate Item", price=500, stock=1)
    with pytest.raises(ErrSKUConflict):
        await svc.create(req)

    assert len(producer.published_events) == 0


@pytest.mark.asyncio
async def test_product_service_cache_aside_get_by_id() -> None:
    repo = MockRepository()
    cache = MockCache()
    producer = MockProducer()
    svc = ProductService(repo, cache, producer)

    product_id = uuid.uuid4()
    repo.products[str(product_id)] = Product(
        id=product_id,
        sku="CACHE-001",
        name="Cached Item",
        price=1200,
        stock=5,
        version=1,
    )

    # 1st Call: Cache Miss -> should query DB (find_by_id_calls == 1) and populate Redis cache
    p1 = await svc.get_by_id(str(product_id))
    assert p1.sku == "CACHE-001"
    assert repo.find_by_id_calls == 1

    # 2nd Call: Cache Hit -> should return from Redis WITHOUT querying DB again (find_by_id_calls stays 1)
    p2 = await svc.get_by_id(str(product_id))
    assert p2.sku == "CACHE-001"
    assert repo.find_by_id_calls == 1


@pytest.mark.asyncio
async def test_product_service_patch_stock_optimistic_locking() -> None:
    repo = MockRepository()
    cache = MockCache()
    producer = MockProducer()
    svc = ProductService(repo, cache, producer)

    created = await svc.create(
        CreateProductRequest(sku="MOUSE-01", name="Wireless Mouse", price=990, stock=20)
    )
    producer.published_events.clear()

    # 1. Success - updates stock, refreshes cache, and increments version from 1 to 2
    updated = await svc.patch_stock(str(created.id), PatchStockRequest(stock=15))
    assert updated.stock == 15
    assert updated.version == 2

    cached = await cache.get(str(created.id))
    assert cached is not None and cached.stock == 15
    assert len(producer.published_events) == 1

    # 2. Fail - negative stock returns ErrNegativeStock
    with pytest.raises(ErrNegativeStock):
        await svc.patch_stock(str(created.id), PatchStockRequest(stock=-5))

    # 3. Fail - concurrent modification returns ErrVersionConflict
    repo.force_conflict_err = True
    with pytest.raises(ErrVersionConflict):
        await svc.patch_stock(str(created.id), PatchStockRequest(stock=10))
