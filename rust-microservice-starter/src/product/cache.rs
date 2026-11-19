use crate::platform::cache::RedisCache;
use crate::product::model::{PaginatedProducts, Product};
use anyhow::Result;
use async_trait::async_trait;
use std::sync::Arc;
use std::time::Duration;
use tracing::{info, warn};

/// Cache defines the distributed caching contract for products.
#[async_trait]
pub trait Cache: Send + Sync {
    async fn get(&self, id: &str) -> Result<Option<Product>>;
    async fn set(&self, product: &Product) -> Result<()>;
    async fn delete(&self, id: &str) -> Result<()>;
    async fn get_list(&self, page: i64, limit: i64) -> Result<Option<PaginatedProducts>>;
    async fn set_list(&self, page: i64, limit: i64, list: &PaginatedProducts) -> Result<()>;
    async fn invalidate_lists(&self) -> Result<()>;
}

pub struct RedisProductCache {
    redis: Option<RedisCache>,
    ttl: Duration,
}

pub fn new_cache(redis: Option<RedisCache>, ttl: Duration) -> Arc<dyn Cache> {
    Arc::new(RedisProductCache { redis, ttl })
}

pub fn cache_key(id: &str) -> String {
    format!("product:{}", id)
}

pub fn list_cache_key(page: i64, limit: i64) -> String {
    format!("products:list:page:{}:limit:{}", page, limit)
}

#[async_trait]
impl Cache for RedisProductCache {
    async fn get(&self, id: &str) -> Result<Option<Product>> {
        let Some(ref redis) = self.redis else {
            return Ok(None);
        };

        match redis.get(&cache_key(id)).await {
            Ok(Some(bytes)) => {
                let product: Product = serde_json::from_slice(&bytes)?;
                info!(id = %id, sku = %product.sku, "⚡ [Redis Cache HIT]");
                Ok(Some(product))
            }
            Ok(None) => Ok(None),
            Err(err) => {
                warn!(id = %id, error = %err, "Redis cache get failed");
                Err(err)
            }
        }
    }

    async fn set(&self, product: &Product) -> Result<()> {
        let Some(ref redis) = self.redis else {
            return Ok(());
        };

        let data = serde_json::to_vec(product)?;
        let key = cache_key(&product.id.to_string());
        if let Err(err) = redis.set(&key, &data, self.ttl).await {
            warn!(id = %product.id, error = %err, "Redis cache set failed");
            return Err(err);
        }
        Ok(())
    }

    async fn delete(&self, id: &str) -> Result<()> {
        let Some(ref redis) = self.redis else {
            return Ok(());
        };

        if let Err(err) = redis.del(&cache_key(id)).await {
            warn!(id = %id, error = %err, "Redis cache delete failed");
            return Err(err);
        }
        Ok(())
    }

    async fn get_list(&self, page: i64, limit: i64) -> Result<Option<PaginatedProducts>> {
        let Some(ref redis) = self.redis else {
            return Ok(None);
        };

        let key = list_cache_key(page, limit);
        match redis.get(&key).await {
            Ok(Some(bytes)) => {
                let list: PaginatedProducts = serde_json::from_slice(&bytes)?;
                info!(page = page, limit = limit, "⚡ [Redis Cache HIT - Product List]");
                Ok(Some(list))
            }
            Ok(None) => Ok(None),
            Err(err) => {
                warn!(key = %key, error = %err, "Redis cache get list failed");
                Err(err)
            }
        }
    }

    async fn set_list(&self, page: i64, limit: i64, list: &PaginatedProducts) -> Result<()> {
        let Some(ref redis) = self.redis else {
            return Ok(());
        };

        let data = serde_json::to_vec(list)?;
        let key = list_cache_key(page, limit);
        if let Err(err) = redis.set(&key, &data, self.ttl).await {
            warn!(key = %key, error = %err, "Redis cache set list failed");
            return Err(err);
        }
        Ok(())
    }

    async fn invalidate_lists(&self) -> Result<()> {
        let Some(ref redis) = self.redis else {
            return Ok(());
        };

        if let Err(err) = redis.del_pattern("products:list:*").await {
            warn!(error = %err, "Redis cache list invalidation failed");
            return Err(err);
        }
        Ok(())
    }
}
