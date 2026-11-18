from __future__ import annotations

import uuid
from datetime import datetime, timezone
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field
from sqlalchemy import DateTime, Integer, Numeric, String, func
from sqlalchemy.dialects.postgresql import UUID as PG_UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.platform.database import Base

QUEUE_STOCK_CHANGED = "inventory.stock_changed"


# ==================== Domain Exceptions ====================
class ProductError(Exception):
    pass


class ErrProductNotFound(ProductError):
    def __init__(self) -> None:
        super().__init__("product not found")


class ErrInvalidPayload(ProductError):
    def __init__(self) -> None:
        super().__init__("invalid payload: check fields and ensure price/stock >= 0")


class ErrNegativeStock(ProductError):
    def __init__(self) -> None:
        super().__init__("stock cannot be negative")


class ErrVersionConflict(ProductError):
    def __init__(self) -> None:
        super().__init__("conflict: product was modified by another request")


class ErrSKUConflict(ProductError):
    def __init__(self) -> None:
        super().__init__("SKU might already exist or database error")


# ==================== SQLAlchemy 2.0 ORM Entity ====================
class ProductModel(Base):
    __tablename__ = "products"

    id: Mapped[uuid.UUID] = mapped_column(
        PG_UUID(as_uuid=True),
        primary_key=True,
        default=uuid.uuid4,
    )
    sku: Mapped[str] = mapped_column(String(64), unique=True, nullable=False)
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    price: Mapped[float] = mapped_column(Numeric(12, 2, asdecimal=False), nullable=False)
    stock: Mapped[int] = mapped_column(Integer, nullable=False, default=0)
    version: Mapped[int] = mapped_column(Integer, nullable=False, default=1)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        nullable=False,
        server_default=func.now(),
        default=lambda: datetime.now(timezone.utc),
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        nullable=False,
        server_default=func.now(),
        default=lambda: datetime.now(timezone.utc),
    )
    deleted_at: Mapped[Optional[datetime]] = mapped_column(
        DateTime(timezone=True),
        nullable=True,
        index=True,
    )


# ==================== Pydantic v2 DTOs & Domain Models ====================
class Product(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: uuid.UUID = Field(default_factory=uuid.uuid4)
    sku: str
    name: str
    price: float
    stock: int = 0
    version: int = 1
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    updated_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))


class CreateProductRequest(BaseModel):
    sku: str = ""
    name: str = ""
    price: float = 0.0
    stock: int = 0


class PatchStockRequest(BaseModel):
    stock: int = 0


class StockChangedEvent(BaseModel):
    event_id: str
    product_id: uuid.UUID
    sku: str
    new_stock: int
    timestamp: datetime
