from __future__ import annotations

import uuid
from abc import ABC, abstractmethod
from datetime import datetime, timezone

from sqlalchemy import func, select, update
from sqlalchemy.exc import IntegrityError
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from app.product.model import (
    ErrProductNotFound,
    ErrSKUConflict,
    ErrVersionConflict,
    Product,
    ProductModel,
)


class Repository(ABC):
    """Defines the data access contract for product persistence."""

    @abstractmethod
    async def create(self, product: Product) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def find_all(self, page: int, limit: int) -> tuple[list[Product], int]:
        raise NotImplementedError

    @abstractmethod
    async def find_by_id(self, product_id: str) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def update(self, product: Product) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def update_stock_with_version(self, product: Product, new_stock: int) -> Product:
        raise NotImplementedError

    @abstractmethod
    async def delete(self, product_id: str) -> None:
        raise NotImplementedError


class SQLAlchemyRepository(Repository):
    """SQLAlchemy 2.0 Async-backed Product Repository."""

    def __init__(self, session_factory: async_sessionmaker[AsyncSession]) -> None:
        self._session_factory = session_factory

    async def create(self, product: Product) -> Product:
        now = datetime.now(timezone.utc)
        entity = ProductModel(
            id=product.id or uuid.uuid4(),
            sku=product.sku,
            name=product.name,
            price=product.price,
            stock=product.stock,
            version=1,
            created_at=now,
            updated_at=now,
        )
        async with self._session_factory() as session:
            try:
                session.add(entity)
                await session.commit()
                await session.refresh(entity)
                return Product.model_validate(entity)
            except IntegrityError as exc:
                await session.rollback()
                raise ErrSKUConflict() from exc

    async def find_all(self, page: int, limit: int) -> tuple[list[Product], int]:
        offset = (page - 1) * limit
        async with self._session_factory() as session:
            count_stmt = (
                select(func.count())
                .select_from(ProductModel)
                .where(ProductModel.deleted_at.is_(None))
            )
            total = int((await session.execute(count_stmt)).scalar_one())

            stmt = (
                select(ProductModel)
                .where(ProductModel.deleted_at.is_(None))
                .order_by(ProductModel.created_at.desc())
                .limit(limit)
                .offset(offset)
            )
            result = await session.execute(stmt)
            rows = result.scalars().all()
            return [Product.model_validate(row) for row in rows], total

    async def find_by_id(self, product_id: str) -> Product:
        try:
            parsed_id = uuid.UUID(product_id)
        except ValueError as exc:
            raise ErrProductNotFound() from exc

        async with self._session_factory() as session:
            stmt = select(ProductModel).where(
                ProductModel.id == parsed_id,
                ProductModel.deleted_at.is_(None),
            )
            result = await session.execute(stmt)
            entity = result.scalar_one_or_none()
            if entity is None:
                raise ErrProductNotFound()
            return Product.model_validate(entity)

    async def update(self, product: Product) -> Product:
        now = datetime.now(timezone.utc)
        async with self._session_factory() as session:
            stmt = (
                update(ProductModel)
                .where(
                    ProductModel.id == product.id,
                    ProductModel.deleted_at.is_(None),
                )
                .values(
                    sku=product.sku,
                    name=product.name,
                    price=product.price,
                    stock=product.stock,
                    version=product.version,
                    updated_at=now,
                )
            )
            await session.execute(stmt)
            await session.commit()
            product.updated_at = now
            return product

    async def update_stock_with_version(self, product: Product, new_stock: int) -> Product:
        """Updates product stock using optimistic locking via the version column."""
        now = datetime.now(timezone.utc)
        next_version = product.version + 1
        async with self._session_factory() as session:
            stmt = (
                update(ProductModel)
                .where(
                    ProductModel.id == product.id,
                    ProductModel.version == product.version,
                    ProductModel.deleted_at.is_(None),
                )
                .values(
                    stock=new_stock,
                    version=next_version,
                    updated_at=now,
                )
            )
            result = await session.execute(stmt)
            await session.commit()

            if result.rowcount == 0:
                raise ErrVersionConflict()

            product.stock = new_stock
            product.version = next_version
            product.updated_at = now
            return product

    async def delete(self, product_id: str) -> None:
        try:
            parsed_id = uuid.UUID(product_id)
        except ValueError as exc:
            raise ErrProductNotFound() from exc

        now = datetime.now(timezone.utc)
        async with self._session_factory() as session:
            stmt = (
                update(ProductModel)
                .where(
                    ProductModel.id == parsed_id,
                    ProductModel.deleted_at.is_(None),
                )
                .values(deleted_at=now)
            )
            result = await session.execute(stmt)
            await session.commit()

            if result.rowcount == 0:
                raise ErrProductNotFound()
