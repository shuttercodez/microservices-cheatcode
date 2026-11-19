from __future__ import annotations

import uuid
from abc import ABC, abstractmethod
from datetime import datetime, timezone

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from app.order.model import ErrOrderNotFound, Order, OrderModel


class Repository(ABC):
    """Defines the data access contract for order persistence."""

    @abstractmethod
    async def create(self, order: Order) -> Order:
        raise NotImplementedError

    @abstractmethod
    async def find_all(self, page: int, limit: int) -> tuple[list[Order], int]:
        raise NotImplementedError

    @abstractmethod
    async def find_by_id(self, order_id: str) -> Order:
        raise NotImplementedError


class SQLAlchemyRepository(Repository):
    """SQLAlchemy 2.0 Async-backed Order Repository."""

    def __init__(self, session_factory: async_sessionmaker[AsyncSession]) -> None:
        self._session_factory = session_factory

    async def create(self, order: Order) -> Order:
        now = datetime.now(timezone.utc)
        entity = OrderModel(
            id=order.id or uuid.uuid4(),
            product_id=order.product_id,
            quantity=order.quantity,
            total_price=order.total_price,
            status=order.status,
            created_at=now,
            updated_at=now,
        )
        async with self._session_factory() as session:
            session.add(entity)
            await session.commit()
            await session.refresh(entity)
            return Order.model_validate(entity)

    async def find_all(self, page: int, limit: int) -> tuple[list[Order], int]:
        offset = (page - 1) * limit
        async with self._session_factory() as session:
            count_stmt = (
                select(func.count())
                .select_from(OrderModel)
                .where(OrderModel.deleted_at.is_(None))
            )
            total = int((await session.execute(count_stmt)).scalar_one())

            stmt = (
                select(OrderModel)
                .where(OrderModel.deleted_at.is_(None))
                .order_by(OrderModel.created_at.desc())
                .limit(limit)
                .offset(offset)
            )
            result = await session.execute(stmt)
            rows = result.scalars().all()
            return [Order.model_validate(row) for row in rows], total

    async def find_by_id(self, order_id: str) -> Order:
        try:
            parsed_id = uuid.UUID(order_id)
        except ValueError as exc:
            raise ErrOrderNotFound() from exc

        async with self._session_factory() as session:
            stmt = select(OrderModel).where(
                OrderModel.id == parsed_id,
                OrderModel.deleted_at.is_(None),
            )
            result = await session.execute(stmt)
            entity = result.scalar_one_or_none()
            if entity is None:
                raise ErrOrderNotFound()
            return Order.model_validate(entity)
