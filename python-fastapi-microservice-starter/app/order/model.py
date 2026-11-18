from __future__ import annotations

import uuid
from datetime import datetime, timezone
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field
from sqlalchemy import DateTime, ForeignKey, Integer, Numeric, String, func
from sqlalchemy.dialects.postgresql import UUID as PG_UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.platform.database import Base

QUEUE_ORDER_CREATED = "order.created"


# ==================== Domain Exceptions ====================
class OrderError(Exception):
    pass


class ErrOrderNotFound(OrderError):
    def __init__(self) -> None:
        super().__init__("order not found")


class ErrInvalidOrderInput(OrderError):
    def __init__(self) -> None:
        super().__init__("invalid order payload: product_id is required and quantity must be > 0")


class ErrInsufficientStock(OrderError):
    def __init__(self) -> None:
        super().__init__("insufficient product stock for this order")


# ==================== SQLAlchemy 2.0 ORM Entity ====================
class OrderModel(Base):
    __tablename__ = "orders"

    id: Mapped[uuid.UUID] = mapped_column(
        PG_UUID(as_uuid=True),
        primary_key=True,
        default=uuid.uuid4,
    )
    product_id: Mapped[uuid.UUID] = mapped_column(
        PG_UUID(as_uuid=True),
        ForeignKey("products.id"),
        nullable=False,
        index=True,
    )
    quantity: Mapped[int] = mapped_column(Integer, nullable=False)
    total_price: Mapped[float] = mapped_column(Numeric(12, 2, asdecimal=False), nullable=False)
    status: Mapped[str] = mapped_column(String(32), nullable=False, default="COMPLETED")
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
class Order(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: uuid.UUID = Field(default_factory=uuid.uuid4)
    product_id: uuid.UUID
    quantity: int
    total_price: float
    status: str = "COMPLETED"
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    updated_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))


class CreateOrderRequest(BaseModel):
    product_id: str = ""
    quantity: int = 0


class OrderCreatedEvent(BaseModel):
    event_id: str
    order_id: uuid.UUID
    product_id: uuid.UUID
    quantity: int
    total_price: float
    timestamp: datetime
