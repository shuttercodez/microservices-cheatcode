use crate::platform::queue::RabbitMQ;
use crate::product::model::{Product, StockChangedEvent, QUEUE_STOCK_CHANGED};
use anyhow::Result;
use async_trait::async_trait;
use chrono::Utc;
use std::sync::Arc;
use tracing::{error, info};
use uuid::Uuid;

/// Producer defines the interface for publishing product domain events to a message broker.
#[async_trait]
pub trait Producer: Send + Sync {
    async fn publish_stock_changed(&self, product: &Product) -> Result<()>;
}

pub struct RabbitMQProducer {
    mq: Option<RabbitMQ>,
}

pub fn new_producer(mq: Option<RabbitMQ>) -> Arc<dyn Producer> {
    Arc::new(RabbitMQProducer { mq })
}

#[async_trait]
impl Producer for RabbitMQProducer {
    async fn publish_stock_changed(&self, product: &Product) -> Result<()> {
        let Some(ref mq) = self.mq else {
            return Ok(());
        };

        let event = StockChangedEvent {
            event_id: Uuid::new_v4().to_string(),
            product_id: product.id,
            sku: product.sku.clone(),
            new_stock: product.stock,
            timestamp: Utc::now(),
        };

        let body = match serde_json::to_vec(&event) {
            Ok(b) => b,
            Err(err) => {
                error!(error = %err, "Failed to marshal stock event");
                return Err(err.into());
            }
        };

        if let Err(err) = mq.publish(QUEUE_STOCK_CHANGED, &body).await {
            error!(error = %err, "Failed to publish message to RabbitMQ");
            return Err(err);
        }

        info!(
            event_id = %event.event_id,
            sku = %product.sku,
            stock = product.stock,
            "📤 [Producer Published Event]"
        );
        Ok(())
    }
}
