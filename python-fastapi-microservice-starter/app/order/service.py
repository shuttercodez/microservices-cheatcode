from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Optional

from app.order.cache import Cache
from app.order.model import (
    CreateOrderRequest,
    ErrInsufficientStock,
    ErrInvalidOrderInput,
    Order,
    PaginatedOrders,
)
from app.order.producer import Producer
from app.order.repository import Repository
from app.product.model import PatchStockRequest
from app.product.service import Service as ProductService


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
    """Defines the business logic contract for managing orders."""

    @abstractmethod
    async def create_order(self, req: CreateOrderRequest) -> Order:
        raise NotImplementedError

    @abstractmethod
    async def get_all_orders(self, page: int = 1, limit: int = 10) -> PaginatedOrders:
        raise NotImplementedError

    @abstractmethod
    async def get_order_by_id(self, order_id: str) -> Order:
        raise NotImplementedError


class OrderService(Service):
    """Order Service with injected OrderRepository, OrderCache, ProductService, and OrderProducer."""

    def __init__(
        self,
        order_repo: Repository,
        order_cache: Optional[Cache],
        product_service: ProductService,
        producer: Producer,
    ) -> None:
        self._order_repo = order_repo
        self._order_cache = order_cache
        self._product_service = product_service
        self._producer = producer

    async def create_order(self, req: CreateOrderRequest) -> Order:
        """Verifies product availability, deducts stock via ProductService, persists the order, caches it, and publishes an event."""
        if not req.product_id or req.quantity <= 0:
            raise ErrInvalidOrderInput()

        # 1. Fetch product details and verify stock availability via ProductService
        prod = await self._product_service.get_by_id(req.product_id)

        if prod.stock < req.quantity:
            raise ErrInsufficientStock()

        # 2. Deduct stock via ProductService.patch_stock (enforcing optimistic locking)
        new_stock = prod.stock - req.quantity
        await self._product_service.patch_stock(
            req.product_id,
            PatchStockRequest(stock=new_stock),
        )

        # 3. Calculate total price and persist the order record
        order = Order(
            product_id=prod.id,
            quantity=req.quantity,
            total_price=prod.price * float(req.quantity),
            status="COMPLETED",
        )
        created = await self._order_repo.create(order)

        # 4. Populate Redis cache and invalidate cached order lists
        if self._order_cache is not None:
            await self._order_cache.set(created)
            await self._order_cache.invalidate_lists()

        # 5. Publish order created event to RabbitMQ
        await self._producer.publish_order_created(created)

        return created

    async def get_all_orders(self, page: int = 1, limit: int = 10) -> PaginatedOrders:
        page, limit = _normalize_pagination(page, limit)

        if self._order_cache is not None:
            cached = await self._order_cache.get_list(page, limit)
            if cached is not None:
                return cached

        orders, total = await self._order_repo.find_all(page, limit)
        res = PaginatedOrders(
            data=orders,
            page=page,
            limit=limit,
            total=total,
            total_pages=_calc_total_pages(total, limit),
        )

        if self._order_cache is not None:
            await self._order_cache.set_list(page, limit, res)
        return res

    async def get_order_by_id(self, order_id: str) -> Order:
        if self._order_cache is not None:
            cached = await self._order_cache.get(order_id)
            if cached is not None:
                return cached

        order = await self._order_repo.find_by_id(order_id)

        if self._order_cache is not None:
            await self._order_cache.set(order)
        return order
