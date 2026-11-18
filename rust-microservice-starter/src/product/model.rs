use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use thiserror::Error;
use uuid::Uuid;

pub const QUEUE_STOCK_CHANGED: &str = "inventory.stock_changed";

#[derive(Debug, Error, PartialEq, Eq, Clone)]
pub enum ProductError {
    #[error("product not found")]
    NotFound,
    #[error("invalid payload: check fields and ensure price/stock >= 0")]
    InvalidPayload,
    #[error("stock cannot be negative")]
    NegativeStock,
    #[error("conflict: product was modified by another request")]
    VersionConflict,
    #[error("SKU might already exist or database error")]
    SkuConflict,
    #[error("internal error: {0}")]
    Internal(String),
}

/// Product represents the database entity for products.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, sqlx::FromRow)]
pub struct Product {
    pub id: Uuid,
    pub sku: String,
    pub name: String,
    pub price: f64,
    pub stock: i32,
    pub version: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    #[serde(skip)]
    pub deleted_at: Option<DateTime<Utc>>,
}

/// CreateProductRequest represents the request DTO for creating or replacing a product (POST / PUT).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateProductRequest {
    pub sku: String,
    pub name: String,
    pub price: f64,
    pub stock: i32,
}

/// PatchStockRequest represents the request DTO for updating product stock (PATCH).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatchStockRequest {
    pub stock: i32,
}

/// StockChangedEvent represents the event payload published to RabbitMQ when product stock changes.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StockChangedEvent {
    pub event_id: String,
    pub product_id: Uuid,
    pub sku: String,
    pub new_stock: i32,
    pub timestamp: DateTime<Utc>,
}
