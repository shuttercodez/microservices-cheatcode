use crate::product::model::StockChangedEvent;
use futures_util::StreamExt;
use lapin::options::{BasicAckOptions, BasicConsumeOptions};
use lapin::types::FieldTable;
use lapin::Connection;
use std::sync::Arc;
use tokio::sync::watch;
use tracing::{error, info};

/// Runs a background worker to consume stock change events from RabbitMQ using `lapin`.
pub async fn start_queue_consumer(
    mut shutdown_rx: watch::Receiver<bool>,
    conn: Arc<Connection>,
    queue_name: &str,
) {
    let channel = match conn.create_channel().await {
        Ok(ch) => ch,
        Err(err) => {
            error!(error = %err, "Consumer failed to open channel");
            return;
        }
    };

    let mut consumer = match channel
        .basic_consume(
            queue_name,
            "inventory-worker",
            BasicConsumeOptions::default(),
            FieldTable::default(),
        )
        .await
    {
        Ok(c) => c,
        Err(err) => {
            error!(error = %err, "Failed to register consumer");
            return;
        }
    };

    info!("RabbitMQ Consumer started waiting for messages...");
    loop {
        tokio::select! {
            _ = shutdown_rx.changed() => {
                info!("Stopping RabbitMQ consumer...");
                let _ = channel.close(200, "Consumer shutdown").await;
                return;
            }
            delivery = consumer.next() => {
                match delivery {
                    Some(Ok(delivery)) => {
                        if let Ok(event) = serde_json::from_slice::<StockChangedEvent>(&delivery.data) {
                            info!(
                                event_id = %event.event_id,
                                sku = %event.sku,
                                new_stock = event.new_stock,
                                "📥 [Axum Consumer Received Event]"
                            );
                            let _ = delivery.ack(BasicAckOptions::default()).await;
                        }
                    }
                    Some(Err(err)) => {
                        error!(error = %err, "Error receiving delivery from RabbitMQ");
                        return;
                    }
                    None => return,
                }
            }
        }
    }
}
