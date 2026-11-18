from __future__ import annotations

import uuid
from typing import Optional

import pytest

from app.order import (
    CreateOrderRequest,
    ErrInsufficientStock,
    ErrOrderNotFound,
    Order,
    OrderService,
    Producer,
    Repository,
)
from app.product import (
    CreateProductRequest,
    ErrProductNotFound,
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

    async def find_all(self) -> list[Order]:
        return [o.model_copy(deep=True) for o in self.orders.values()]

    async def find_by_id(self, order_id: str) -> Order:
        o = self.orders.get(order_id)
        if o is None:
            raise ErrOrderNotFound()
        return o.model_copy(deep=True)


class MockProductService(ProductService):
    def __init__(self) -> None:
        self.products: dict[str, Product] = {}
        self.patch_stock_called: bool = False
        self.force_patch_err: Optional[Exception] = None

    async def create(self, req: CreateProductRequest) -> Product:
        raise NotImplementedError

    async def get_all(self) -> list[Product]:
        return []

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

    svc = OrderService(order_repo, prod_svc, producer)

    order = await svc.create_order(
        CreateOrderRequest(product_id=str(product_id), quantity=3)
    )

    assert order.total_price == 4500
    assert prod_svc.products[str(product_id)].stock == 7
    assert len(producer.published_orders) == 1


@pytest.mark.asyncio
async def test_order_service_create_order_insufficient_stock_fails() -> None:
    product_id = uuid.uuid4()
    order_repo = MockOrderRepository()
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

    svc = OrderService(order_repo, prod_svc, producer)

    with pytest.raises(ErrInsufficientStock):
        await svc.create_order(
            CreateOrderRequest(product_id=str(product_id), quantity=5)
        )

    assert not prod_svc.patch_stock_called
    assert len(producer.published_orders) == 0


@pytest.mark.asyncio
async def test_order_service_create_order_product_not_found_fails() -> None:
    order_repo = MockOrderRepository()
    prod_svc = MockProductService()
    producer = MockOrderProducer()
    svc = OrderService(order_repo, prod_svc, producer)

    with pytest.raises(ErrProductNotFound):
        await svc.create_order(
            CreateOrderRequest(product_id=str(uuid.uuid4()), quantity=1)
        )
