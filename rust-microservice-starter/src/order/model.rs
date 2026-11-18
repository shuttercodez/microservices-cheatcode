use crate::product::ProductError;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use thiserror::Error;
use uuid::Uuid;

pub const QUEUE_ORDER_CREATED: &str = "order.created";

#[derive(Debug, Error, PartialEq, Eq, Clone)]
pub enum OrderError {
    #[error("order not found")]
    NotFound,
    #[error("invalid order payload: product_id is required and quantity must be > 0")]
    InvalidInput,
    #[error("insufficient product stock for this order")]
    InsufficientStock,
    #[error("{0}")]
    Product(#[from] ProductError),
    #[error("internal error: {0}")]
    Internal(String),
}

/// Order represents the database entity for customer orders.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, sqlx::FromRow)]
pub struct Order {
    pub id: Uuid,
    pub product_id: Uuid,
    pub quantity: i32,
    pub total_price: f64,
    pub status: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    #[serde(skip)]
    pub deleted_at: Option<DateTime<Utc>>,
}

/// CreateOrderRequest represents the request DTO for creating a new order (POST /api/v1/orders).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateOrderRequest {
    pub product_id: String,
    pub quantity: i32,
}

/// OrderCreatedEvent represents the event payload published to RabbitMQ when an order is created.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderCreatedEvent {
    pub event_id: String,
    pub order_id: Uuid,
    pub product_id: Uuid,
    pub quantity: i32,
    pub total_price: f64,
    pub timestamp: DateTime<Utc>,
}
