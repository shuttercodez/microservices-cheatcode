use crate::order::model::{Order, OrderError};
use async_trait::async_trait;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

/// Repository defines the data access contract for order persistence.
#[async_trait]
pub trait Repository: Send + Sync {
    async fn create(&self, order: &mut Order) -> Result<(), OrderError>;
    async fn find_all(&self, page: i64, limit: i64) -> Result<(Vec<Order>, i64), OrderError>;
    async fn find_by_id(&self, id: &str) -> Result<Order, OrderError>;
}

pub struct SqlxRepository {
    db: PgPool,
}

pub fn new_repository(db: PgPool) -> Arc<dyn Repository> {
    Arc::new(SqlxRepository { db })
}

#[async_trait]
impl Repository for SqlxRepository {
    async fn create(&self, order: &mut Order) -> Result<(), OrderError> {
        let row: Order = sqlx::query_as(
            r#"
            INSERT INTO orders (product_id, quantity, total_price, status)
            VALUES ($1, $2, $3::float8::numeric, $4)
            RETURNING id, product_id, quantity, total_price::float8 AS total_price, status, created_at, updated_at, deleted_at
            "#,
        )
        .bind(order.product_id)
        .bind(order.quantity)
        .bind(order.total_price)
        .bind(&order.status)
        .fetch_one(&self.db)
        .await
        .map_err(|e| OrderError::Internal(e.to_string()))?;

        *order = row;
        Ok(())
    }

    async fn find_all(&self, page: i64, limit: i64) -> Result<(Vec<Order>, i64), OrderError> {
        let (total,): (i64,) =
            sqlx::query_as("SELECT COUNT(*) FROM orders WHERE deleted_at IS NULL")
                .fetch_one(&self.db)
                .await
                .map_err(|e| OrderError::Internal(e.to_string()))?;

        let offset = (page - 1) * limit;
        let orders: Vec<Order> = sqlx::query_as(
            r#"
            SELECT id, product_id, quantity, total_price::float8 AS total_price, status, created_at, updated_at, deleted_at
            FROM orders
            WHERE deleted_at IS NULL
            ORDER BY created_at DESC
            LIMIT $1 OFFSET $2
            "#,
        )
        .bind(limit)
        .bind(offset)
        .fetch_all(&self.db)
        .await
        .map_err(|e| OrderError::Internal(e.to_string()))?;

        Ok((orders, total))
    }

    async fn find_by_id(&self, id: &str) -> Result<Order, OrderError> {
        let parsed_id = Uuid::parse_str(id).map_err(|_| OrderError::NotFound)?;

        let order: Option<Order> = sqlx::query_as(
            r#"
            SELECT id, product_id, quantity, total_price::float8 AS total_price, status, created_at, updated_at, deleted_at
            FROM orders
            WHERE id = $1 AND deleted_at IS NULL
            "#,
        )
        .bind(parsed_id)
        .fetch_optional(&self.db)
        .await
        .map_err(|e| OrderError::Internal(e.to_string()))?;

        order.ok_or(OrderError::NotFound)
    }
}
