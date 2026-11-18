from __future__ import annotations

from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncEngine, AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.orm import DeclarativeBase


class Base(DeclarativeBase):
    pass


class PostgresDatabase:
    """Manages the SQLAlchemy 2.0 AsyncEngine and AsyncSession factory."""

    def __init__(self, engine: AsyncEngine) -> None:
        self.engine = engine
        self.session_factory = async_sessionmaker(
            bind=engine,
            class_=AsyncSession,
            expire_on_commit=False,
        )

    @classmethod
    async def connect(cls, dsn: str) -> PostgresDatabase:
        engine = create_async_engine(dsn, pool_size=10, max_overflow=20, pool_pre_ping=True)
        db = cls(engine)
        await db.ping()
        return db

    async def ping(self) -> None:
        async with self.engine.connect() as conn:
            await conn.execute(text("SELECT 1"))

    async def close(self) -> None:
        await self.engine.dispose()
