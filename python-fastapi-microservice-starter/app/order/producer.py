from __future__ import annotations

import logging
import uuid
from abc import ABC, abstractmethod
from datetime import datetime, timezone
from typing import Optional

from app.order.model import QUEUE_ORDER_CREATED, Order, OrderCreatedEvent
from app.platform.queue import RabbitMQ

logger = logging.getLogger(__name__)


class Producer(ABC):
    """Defines the interface for publishing order domain events to a message broker."""

    @abstractmethod
    async def publish_order_created(self, order: Order) -> None:
        raise NotImplementedError


class RabbitMQProducer(Producer):
    """RabbitMQ-backed order event producer."""

    def __init__(self, mq: Optional[RabbitMQ]) -> None:
        self._mq = mq

    async def publish_order_created(self, order: Order) -> None:
        if self._mq is None:
            return

        event = OrderCreatedEvent(
            event_id=str(uuid.uuid4()),
            order_id=order.id,
            product_id=order.product_id,
            quantity=order.quantity,
            total_price=order.total_price,
            timestamp=datetime.now(timezone.utc),
        )

        try:
            body = event.model_dump_json().encode("utf-8")
            await self._mq.publish(QUEUE_ORDER_CREATED, body)
            logger.info(
                "📤 [Order Producer Published Event]",
                extra={
                    "event_id": event.event_id,
                    "order_id": str(order.id),
                    "product_id": str(order.product_id),
                    "total_price": order.total_price,
                },
            )
        except Exception as exc:
            logger.error("Failed to publish order event to RabbitMQ", extra={"error": str(exc)})
            raise
