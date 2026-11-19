from __future__ import annotations

import uuid
from typing import Optional

import pytest

from app.order import (
    Cache as OrderCache,
    CreateOrderRequest,
    ErrInsufficientStock,
    ErrOrderNotFound,
    Order,
    OrderService,
    PaginatedOrders,
    Producer,
    Repository,
    list_cache_key,
)
from app.product import (
    CreateProductRequest,
    ErrProductNotFound,
    PaginatedProducts,
    PatchStockRequest,
    Product,
    Service as ProductService,
)


# ==================== 1. Mock Objects ====================
class MockOrderRepository(Repository):
    def __init__(self) -> None:
        self.orders: dict[str, Order] = {}

    async def create(self, order: Order) -> Order:
        copied = order.model_copy(deep=True)
        if not copied.id:
            copied.id = uuid.uuid4()
        self.orders[str(copied.id)] = copied
        return copied.model_copy(deep=True)

    async def find_all(self, page: int, limit: int) -> tuple[list[Order], int]:
        all_items = [o.model_copy(deep=True) for o in self.orders.values()]
        total = len(all_items)
        offset = (page - 1) * limit
        return all_items[offset : offset + limit], total

    async def find_by_id(self, order_id: str) -> Order:
        o = self.orders.get(order_id)
        if o is None:
            raise ErrOrderNotFound()
        return o.model_copy(deep=True)


class MockOrderCache(OrderCache):
    def __init__(self) -> None:
        self.store: dict[str, Order] = {}
        self.list_store: dict[str, PaginatedOrders] = {}

    async def get(self, order_id: str) -> Optional[Order]:
        o = self.store.get(order_id)
        return o.model_copy(deep=True) if o is not None else None

    async def set(self, order: Order) -> None:
        self.store[str(order.id)] = order.model_copy(deep=True)

    async def get_list(self, page: int, limit: int) -> Optional[PaginatedOrders]:
        key = list_cache_key(page, limit)
        data = self.list_store.get(key)
        return data.model_copy(deep=True) if data is not None else None

    async def set_list(self, page: int, limit: int, list_data: PaginatedOrders) -> None:
        key = list_cache_key(page, limit)
        self.list_store[key] = list_data.model_copy(deep=True)

    async def invalidate_lists(self) -> None:
        self.list_store.clear()


class MockProductService(ProductService):
    def __init__(self) -> None:
        self.products: dict[str, Product] = {}
        self.patch_stock_called: bool = False
        self.force_patch_err: Optional[Exception] = None

    async def create(self, req: CreateProductRequest) -> Product:
        raise NotImplementedError

    async def get_all(self, page: int = 1, limit: int = 10) -> PaginatedProducts:
        return PaginatedProducts(data=[], page=page, limit=limit, total=0, total_pages=0)

    async def get_by_id(self, product_id: str) -> Product:
        p = self.products.get(product_id)
        if p is None:
            raise ErrProductNotFound()
        return p.model_copy(deep=True)

    async def update(self, product_id: str, req: CreateProductRequest) -> Product:
        raise NotImplementedError

    async def patch_stock(self, product_id: str, req: PatchStockRequest) -> Product:
        self.patch_stock_called = True
        if self.force_patch_err is not None:
            raise self.force_patch_err
        p = self.products.get(product_id)
        if p is None:
            raise ErrProductNotFound()
        p.stock = req.stock
        p.version += 1
        return p.model_copy(deep=True)

    async def delete(self, product_id: str) -> None:
        return None


class MockOrderProducer(Producer):
    def __init__(self) -> None:
        self.published_orders: list[Order] = []

    async def publish_order_created(self, order: Order) -> None:
        self.published_orders.append(order.model_copy(deep=True))


# ==================== 2. Unit Tests for Order Service (Cross-Domain) ====================
@pytest.mark.asyncio
async def test_order_service_create_order_success() -> None:
    product_id = uuid.uuid4()
    order_repo = MockOrderRepository()
    order_cache = MockOrderCache()
    prod_svc = MockProductService()
    producer = MockOrderProducer()

    prod_svc.products[str(product_id)] = Product(
        id=product_id,
        sku="KEY-001",
        name="Mechanical Keyboard",
        price=1500,
        stock=10,
        version=1,
    )

    svc = OrderService(order_repo, order_cache, prod_svc, producer)

    order = await svc.create_order(
        CreateOrderRequest(product_id=str(product_id), quantity=3)
    )

    assert order.total_price == 4500
    assert prod_svc.products[str(product_id)].stock == 7
    assert await order_cache.get(str(order.id)) is not None
    assert len(producer.published_orders) == 1


@pytest.mark.asyncio
async def test_order_service_create_order_insufficient_stock_fails() -> None:
    product_id = uuid.uuid4()
    order_repo = MockOrderRepository()
    order_cache = MockOrderCache()
    prod_svc = MockProductService()
    producer = MockOrderProducer()

    prod_svc.products[str(product_id)] = Product(
        id=product_id,
        sku="KEY-001",
        name="Mechanical Keyboard",
        price=1500,
        stock=2,
        version=1,
    )

    svc = OrderService(order_repo, order_cache, prod_svc, producer)

    with pytest.raises(ErrInsufficientStock):
        await svc.create_order(
            CreateOrderRequest(product_id=str(product_id), quantity=5)
        )

    assert not prod_svc.patch_stock_called
    assert len(producer.published_orders) == 0


@pytest.mark.asyncio
async def test_order_service_create_order_product_not_found_fails() -> None:
    order_repo = MockOrderRepository()
    order_cache = MockOrderCache()
    prod_svc = MockProductService()
    producer = MockOrderProducer()
    svc = OrderService(order_repo, order_cache, prod_svc, producer)

    with pytest.raises(ErrProductNotFound):
        await svc.create_order(
            CreateOrderRequest(product_id=str(uuid.uuid4()), quantity=1)
        )
