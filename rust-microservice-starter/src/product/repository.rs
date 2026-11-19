use crate::product::model::{Product, ProductError};
use async_trait::async_trait;
use chrono::Utc;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

/// Repository defines the data access contract for product persistence.
#[async_trait]
pub trait Repository: Send + Sync {
    async fn create(&self, product: &mut Product) -> Result<(), ProductError>;
    async fn find_all(&self, page: i64, limit: i64) -> Result<(Vec<Product>, i64), ProductError>;
    async fn find_by_id(&self, id: &str) -> Result<Product, ProductError>;
    async fn update(&self, product: &mut Product) -> Result<(), ProductError>;
    async fn update_stock_with_version(
        &self,
        product: &mut Product,
        new_stock: i32,
    ) -> Result<(), ProductError>;
    async fn delete(&self, id: &str) -> Result<(), ProductError>;
}

pub struct SqlxRepository {
    db: PgPool,
}

pub fn new_repository(db: PgPool) -> Arc<dyn Repository> {
    Arc::new(SqlxRepository { db })
}

#[async_trait]
impl Repository for SqlxRepository {
    async fn create(&self, product: &mut Product) -> Result<(), ProductError> {
        let row: Product = sqlx::query_as(
            r#"
            INSERT INTO products (sku, name, price, stock, version)
            VALUES ($1, $2, $3::float8::numeric, $4, 1)
            RETURNING id, sku, name, price::float8 AS price, stock, version, created_at, updated_at, deleted_at
            "#,
        )
        .bind(&product.sku)
        .bind(&product.name)
        .bind(product.price)
        .bind(product.stock)
        .fetch_one(&self.db)
        .await
        .map_err(|_| ProductError::SkuConflict)?;

        *product = row;
        Ok(())
    }

    async fn find_all(&self, page: i64, limit: i64) -> Result<(Vec<Product>, i64), ProductError> {
        let (total,): (i64,) =
            sqlx::query_as("SELECT COUNT(*) FROM products WHERE deleted_at IS NULL")
                .fetch_one(&self.db)
                .await
                .map_err(|e| ProductError::Internal(e.to_string()))?;

        let offset = (page - 1) * limit;
        let products: Vec<Product> = sqlx::query_as(
            r#"
            SELECT id, sku, name, price::float8 AS price, stock, version, created_at, updated_at, deleted_at
            FROM products
            WHERE deleted_at IS NULL
            ORDER BY created_at DESC
            LIMIT $1 OFFSET $2
            "#,
        )
        .bind(limit)
        .bind(offset)
        .fetch_all(&self.db)
        .await
        .map_err(|e| ProductError::Internal(e.to_string()))?;

        Ok((products, total))
    }

    async fn find_by_id(&self, id: &str) -> Result<Product, ProductError> {
        let parsed_id = Uuid::parse_str(id).map_err(|_| ProductError::NotFound)?;

        let product: Option<Product> = sqlx::query_as(
            r#"
            SELECT id, sku, name, price::float8 AS price, stock, version, created_at, updated_at, deleted_at
            FROM products
            WHERE id = $1 AND deleted_at IS NULL
            "#,
        )
        .bind(parsed_id)
        .fetch_optional(&self.db)
        .await
        .map_err(|e| ProductError::Internal(e.to_string()))?;

        product.ok_or(ProductError::NotFound)
    }

    async fn update(&self, product: &mut Product) -> Result<(), ProductError> {
        let now = Utc::now();
        sqlx::query(
            r#"
            UPDATE products
            SET sku = $1, name = $2, price = $3::float8::numeric, stock = $4, version = $5, updated_at = $6
            WHERE id = $7 AND deleted_at IS NULL
            "#,
        )
        .bind(&product.sku)
        .bind(&product.name)
        .bind(product.price)
        .bind(product.stock)
        .bind(product.version)
        .bind(now)
        .bind(product.id)
        .execute(&self.db)
        .await
        .map_err(|e| ProductError::Internal(e.to_string()))?;

        product.updated_at = now;
        Ok(())
    }

    async fn update_stock_with_version(
        &self,
        product: &mut Product,
        new_stock: i32,
    ) -> Result<(), ProductError> {
        let now = Utc::now();
        let next_version = product.version + 1;

        let result = sqlx::query(
            r#"
            UPDATE products
            SET stock = $1, version = $2, updated_at = $3
            WHERE id = $4 AND version = $5 AND deleted_at IS NULL
            "#,
        )
        .bind(new_stock)
        .bind(next_version)
        .bind(now)
        .bind(product.id)
        .bind(product.version)
        .execute(&self.db)
        .await
        .map_err(|e| ProductError::Internal(e.to_string()))?;

        if result.rows_affected() == 0 {
            return Err(ProductError::VersionConflict);
        }

        product.stock = new_stock;
        product.version = next_version;
        product.updated_at = now;
        Ok(())
    }

    async fn delete(&self, id: &str) -> Result<(), ProductError> {
        let parsed_id = Uuid::parse_str(id).map_err(|_| ProductError::NotFound)?;

        let result = sqlx::query(
            r#"
            UPDATE products
            SET deleted_at = NOW()
            WHERE id = $1 AND deleted_at IS NULL
            "#,
        )
        .bind(parsed_id)
        .execute(&self.db)
        .await
        .map_err(|e| ProductError::Internal(e.to_string()))?;

        if result.rows_affected() == 0 {
            return Err(ProductError::NotFound);
        }

        Ok(())
    }
}
