from __future__ import annotations

import asyncio
import logging

from aio_pika.abc import AbstractRobustConnection

from app.order.model import OrderCreatedEvent

logger = logging.getLogger(__name__)


async def start_queue_consumer(
    stop_event: asyncio.Event,
    conn: AbstractRobustConnection,
    queue_name: str,
) -> None:
    """Runs a background worker to consume order creation events from RabbitMQ."""
    try:
        channel = await conn.channel()
        queue = await channel.declare_queue(queue_name, durable=True)
    except Exception as exc:
        logger.error("Order consumer failed to open channel", extra={"error": str(exc)})
        return

    logger.info("RabbitMQ Order Consumer started waiting for messages...")
    try:
        async with queue.iterator(consumer_tag="order-worker") as queue_iter:
            async for message in queue_iter:
                if stop_event.is_set():
                    break
                try:
                    event = OrderCreatedEvent.model_validate_json(message.body)
                    logger.info(
                        "📥 [Order Consumer Received Event]",
                        extra={
                            "event_id": event.event_id,
                            "order_id": str(event.order_id),
                            "product_id": str(event.product_id),
                            "quantity": event.quantity,
                            "total_price": event.total_price,
                        },
                    )
                    await message.ack()
                except Exception as exc:
                    logger.warning("Failed to parse order event", extra={"error": str(exc)})
    except asyncio.CancelledError:
        logger.info("Stopping RabbitMQ order consumer...")
    finally:
        if not channel.is_closed:
            await channel.close()
