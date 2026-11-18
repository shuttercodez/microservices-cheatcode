from __future__ import annotations

import aio_pika
from aio_pika import DeliveryMode, Message
from aio_pika.abc import AbstractRobustChannel, AbstractRobustConnection


class RabbitMQ:
    """Manages the async AMQP 0-9-1 connection and channel using aio-pika."""

    def __init__(
        self,
        connection: AbstractRobustConnection,
        channel: AbstractRobustChannel,
    ) -> None:
        self.conn = connection
        self.channel = channel

    @classmethod
    async def connect(cls, url: str) -> RabbitMQ:
        conn = await aio_pika.connect_robust(url)
        channel = await conn.channel()
        return cls(conn, channel)

    async def declare_queue(self, queue_name: str) -> None:
        await self.channel.declare_queue(queue_name, durable=True)

    async def publish(self, queue_name: str, body: bytes) -> None:
        message = Message(
            body=body,
            content_type="application/json",
            delivery_mode=DeliveryMode.PERSISTENT,
        )
        await self.channel.default_exchange.publish(message, routing_key=queue_name)

    def is_closed(self) -> bool:
        return self.conn.is_closed

    async def close(self) -> None:
        if not self.channel.is_closed:
            await self.channel.close()
        if not self.conn.is_closed:
            await self.conn.close()
