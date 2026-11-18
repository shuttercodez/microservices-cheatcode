use anyhow::Result;
use lapin::options::{BasicPublishOptions, QueueDeclareOptions};
use lapin::types::FieldTable;
use lapin::{BasicProperties, Channel, Connection, ConnectionProperties};
use std::sync::Arc;

/// RabbitMQ manages the AMQP 0-9-1 connection and channel using the community-standard `lapin` crate.
#[derive(Clone)]
pub struct RabbitMQ {
    pub conn: Arc<Connection>,
    pub channel: Channel,
}

impl RabbitMQ {
    /// Establishes a connection to RabbitMQ and opens a default publishing channel.
    pub async fn new(url: &str) -> Result<Self> {
        let conn = Connection::connect(url, ConnectionProperties::default()).await?;
        let channel = conn.create_channel().await?;

        Ok(Self {
            conn: Arc::new(conn),
            channel,
        })
    }

    /// Declares a durable queue on the RabbitMQ broker.
    pub async fn declare_queue(&self, queue_name: &str) -> Result<()> {
        let opts = QueueDeclareOptions {
            durable: true,
            ..Default::default()
        };
        self.channel
            .queue_declare(queue_name, opts, FieldTable::default())
            .await?;
        Ok(())
    }

    /// Sends a persistent JSON message (`delivery_mode = 2`) to the specified queue.
    pub async fn publish(&self, queue_name: &str, body: &[u8]) -> Result<()> {
        let props = BasicProperties::default()
            .with_content_type("application/json".into())
            .with_delivery_mode(2); // Persistent (survives broker restarts)

        self.channel
            .basic_publish(
                "",
                queue_name,
                BasicPublishOptions::default(),
                body,
                props,
            )
            .await?
            .await?;

        Ok(())
    }

    /// Checks whether the underlying AMQP connection is closed.
    pub fn is_closed(&self) -> bool {
        !self.conn.status().connected()
    }

    /// Safely closes the AMQP channel and connection.
    pub async fn close(&self) {
        let _ = self.channel.close(200, "Normal shutdown").await;
        let _ = self.conn.close(200, "Normal shutdown").await;
    }
}
