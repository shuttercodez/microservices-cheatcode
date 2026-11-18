use crate::order::model::OrderCreatedEvent;
use futures_util::StreamExt;
use lapin::options::{BasicAckOptions, BasicConsumeOptions};
use lapin::types::FieldTable;
use lapin::Connection;
use std::sync::Arc;
use tokio::sync::watch;
use tracing::{error, info};

/// Runs a background worker to consume order creation events from RabbitMQ using `lapin`.
pub async fn start_queue_consumer(
    mut shutdown_rx: watch::Receiver<bool>,
    conn: Arc<Connection>,
    queue_name: &str,
) {
    let channel = match conn.create_channel().await {
        Ok(ch) => ch,
        Err(err) => {
            error!(error = %err, "Order consumer failed to open channel");
            return;
        }
    };

    let mut consumer = match channel
        .basic_consume(
            queue_name,
            "order-worker",
            BasicConsumeOptions::default(),
            FieldTable::default(),
        )
        .await
    {
        Ok(c) => c,
        Err(err) => {
            error!(error = %err, "Failed to register order consumer");
            return;
        }
    };

    info!("RabbitMQ Order Consumer started waiting for messages...");
    loop {
        tokio::select! {
            _ = shutdown_rx.changed() => {
                info!("Stopping RabbitMQ order consumer...");
                let _ = channel.close(200, "Consumer shutdown").await;
                return;
            }
            delivery = consumer.next() => {
                match delivery {
                    Some(Ok(delivery)) => {
                        if let Ok(event) = serde_json::from_slice::<OrderCreatedEvent>(&delivery.data) {
                            info!(
                                event_id = %event.event_id,
                                order_id = %event.order_id,
                                product_id = %event.product_id,
                                quantity = event.quantity,
                                total_price = event.total_price,
                                "📥 [Order Consumer Received Event]"
                            );
                            let _ = delivery.ack(BasicAckOptions::default()).await;
                        }
                    }
                    Some(Err(err)) => {
                        error!(error = %err, "Error receiving order delivery from RabbitMQ");
                        return;
                    }
                    None => return,
                }
            }
        }
    }
}
