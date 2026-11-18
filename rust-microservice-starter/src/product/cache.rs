use crate::platform::cache::RedisCache;
use crate::product::model::Product;
use anyhow::Result;
use async_trait::async_trait;
use std::sync::Arc;
use std::time::Duration;
use tracing::{info, warn};

const PRODUCT_CACHE_TTL: Duration = Duration::from_secs(5 * 60);

/// Cache defines the distributed caching contract for products.
#[async_trait]
pub trait Cache: Send + Sync {
    async fn get(&self, id: &str) -> Result<Option<Product>>;
    async fn set(&self, product: &Product) -> Result<()>;
    async fn delete(&self, id: &str) -> Result<()>;
}

pub struct RedisProductCache {
    redis: Option<RedisCache>,
}

pub fn new_cache(redis: Option<RedisCache>) -> Arc<dyn Cache> {
    Arc::new(RedisProductCache { redis })
}

fn cache_key(id: &str) -> String {
    format!("product:{}", id)
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
        if let Err(err) = redis.set(&key, &data, PRODUCT_CACHE_TTL).await {
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
}
