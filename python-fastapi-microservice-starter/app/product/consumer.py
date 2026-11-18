from __future__ import annotations

import asyncio
import logging

from aio_pika.abc import AbstractRobustConnection

from app.product.model import StockChangedEvent

logger = logging.getLogger(__name__)


async def start_queue_consumer(
    stop_event: asyncio.Event,
    conn: AbstractRobustConnection,
    queue_name: str,
) -> None:
    """Runs a background worker to consume stock change events from RabbitMQ."""
    try:
        channel = await conn.channel()
        queue = await channel.declare_queue(queue_name, durable=True)
    except Exception as exc:
        logger.error("Consumer failed to open channel", extra={"error": str(exc)})
        return

    logger.info("RabbitMQ Consumer started waiting for messages...")
    try:
        async with queue.iterator(consumer_tag="inventory-worker") as queue_iter:
            async for message in queue_iter:
                if stop_event.is_set():
                    break
                try:
                    event = StockChangedEvent.model_validate_json(message.body)
                    logger.info(
                        "📥 [FastAPI Consumer Received Event]",
                        extra={
                            "event_id": event.event_id,
                            "sku": event.sku,
                            "new_stock": event.new_stock,
                        },
                    )
                    await message.ack()
                except Exception as exc:
                    logger.warning("Failed to parse stock event", extra={"error": str(exc)})
    except asyncio.CancelledError:
        logger.info("Stopping RabbitMQ consumer...")
    finally:
        if not channel.is_closed:
            await channel.close()
