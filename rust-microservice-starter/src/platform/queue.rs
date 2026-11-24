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
    ///
    /// Note: `lapin` interprets a trailing `/` in `amqp://user:pass@host:5672/` as an empty vhost `""`
    /// instead of the default `"/"` vhost (which causes RabbitMQ to reject the handshake unless encoded as `%2f` or omitted).
    /// We normalize a trailing `/` (with no path segments after it) so standard `amqp://...:5672/` URLs work seamlessly.
    pub async fn new(url: &str) -> Result<Self> {
        let normalized_url = Self::normalize_amqp_url(url);
        let conn = Connection::connect(&normalized_url, ConnectionProperties::default()).await?;
        let channel = conn.create_channel().await?;

        Ok(Self {
            conn: Arc::new(conn),
            channel,
        })
    }

    fn normalize_amqp_url(url: &str) -> String {
        if let Some((base, query)) = url.split_once('?') {
            if let Some(stripped) = base.strip_suffix('/') {
                if !stripped.ends_with(":/") {
                    return format!("{}/%2f?{}", stripped, query);
                }
            }
            url.to_string()
        } else if let Some(stripped) = url.strip_suffix('/') {
            // Only replace trailing slash if it immediately follows the host:port (no vhost specified)
            if let Some((_scheme, rest)) = stripped.split_once("://") {
                if !rest.contains('/') {
                    return format!("{}/%2f", stripped);
                }
            }
            url.to_string()
        } else {
            url.to_string()
        }
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
