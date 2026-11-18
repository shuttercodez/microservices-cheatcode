from __future__ import annotations

from abc import ABC, abstractmethod

from app.order.model import (
    CreateOrderRequest,
    ErrInsufficientStock,
    ErrInvalidOrderInput,
    Order,
)
from app.order.producer import Producer
from app.order.repository import Repository
from app.product.model import PatchStockRequest
from app.product.service import Service as ProductService


class Service(ABC):
    """Defines the business logic contract for managing orders."""

    @abstractmethod
    async def create_order(self, req: CreateOrderRequest) -> Order:
        raise NotImplementedError

    @abstractmethod
    async def get_all_orders(self) -> list[Order]:
        raise NotImplementedError

    @abstractmethod
    async def get_order_by_id(self, order_id: str) -> Order:
        raise NotImplementedError


class OrderService(Service):
    """Order Service with injected OrderRepository, ProductService, and OrderProducer."""

    def __init__(
        self,
        order_repo: Repository,
        product_service: ProductService,
        producer: Producer,
    ) -> None:
        self._order_repo = order_repo
        self._product_service = product_service
        self._producer = producer

    async def create_order(self, req: CreateOrderRequest) -> Order:
        """Verifies product availability, deducts stock via ProductService, persists the order, and publishes an event."""
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

        # 4. Publish order created event to RabbitMQ
        await self._producer.publish_order_created(created)

        return created

    async def get_all_orders(self) -> list[Order]:
        return await self._order_repo.find_all()

    async def get_order_by_id(self, order_id: str) -> Order:
        return await self._order_repo.find_by_id(order_id)
