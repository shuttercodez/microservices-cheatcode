from __future__ import annotations

import logging
import uuid
from abc import ABC, abstractmethod
from datetime import datetime, timezone
from typing import Optional

from app.platform.queue import RabbitMQ
from app.product.model import QUEUE_STOCK_CHANGED, Product, StockChangedEvent

logger = logging.getLogger(__name__)


class Producer(ABC):
    """Defines the interface for publishing product domain events to a message broker."""

    @abstractmethod
    async def publish_stock_changed(self, product: Product) -> None:
        raise NotImplementedError


class RabbitMQProducer(Producer):
    """RabbitMQ-backed product event producer."""

    def __init__(self, mq: Optional[RabbitMQ]) -> None:
        self._mq = mq

    async def publish_stock_changed(self, product: Product) -> None:
        if self._mq is None:
            return

        event = StockChangedEvent(
            event_id=str(uuid.uuid4()),
            product_id=product.id,
            sku=product.sku,
            new_stock=product.stock,
            timestamp=datetime.now(timezone.utc),
        )

        try:
            body = event.model_dump_json().encode("utf-8")
            await self._mq.publish(QUEUE_STOCK_CHANGED, body)
            logger.info(
                "📤 [Producer Published Event]",
                extra={
                    "event_id": event.event_id,
                    "sku": product.sku,
                    "stock": product.stock,
                },
            )
        except Exception as exc:
            logger.error("Failed to publish message to RabbitMQ", extra={"error": str(exc)})
            raise
