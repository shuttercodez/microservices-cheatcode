use crate::order::model::{Order, OrderCreatedEvent, QUEUE_ORDER_CREATED};
use crate::platform::queue::RabbitMQ;
use anyhow::Result;
use async_trait::async_trait;
use chrono::Utc;
use std::sync::Arc;
use tracing::{error, info};
use uuid::Uuid;

/// Producer defines the interface for publishing order domain events to a message broker.
#[async_trait]
pub trait Producer: Send + Sync {
    async fn publish_order_created(&self, order: &Order) -> Result<()>;
}

pub struct RabbitMQProducer {
    mq: Option<RabbitMQ>,
}

pub fn new_producer(mq: Option<RabbitMQ>) -> Arc<dyn Producer> {
    Arc::new(RabbitMQProducer { mq })
}

#[async_trait]
impl Producer for RabbitMQProducer {
    async fn publish_order_created(&self, order: &Order) -> Result<()> {
        let Some(ref mq) = self.mq else {
            return Ok(());
        };

        let event = OrderCreatedEvent {
            event_id: Uuid::new_v4().to_string(),
            order_id: order.id,
            product_id: order.product_id,
            quantity: order.quantity,
            total_price: order.total_price,
            timestamp: Utc::now(),
        };

        let body = match serde_json::to_vec(&event) {
            Ok(b) => b,
            Err(err) => {
                error!(error = %err, "Failed to marshal order event");
                return Err(err.into());
            }
        };

        if let Err(err) = mq.publish(QUEUE_ORDER_CREATED, &body).await {
            error!(error = %err, "Failed to publish order event to RabbitMQ");
            return Err(err);
        }

        info!(
            event_id = %event.event_id,
            order_id = %order.id,
            product_id = %order.product_id,
            total_price = order.total_price,
            "📤 [Order Producer Published Event]"
        );
        Ok(())
    }
}
