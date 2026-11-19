use crate::order::model::{Order, PaginatedOrders};
use crate::platform::cache::RedisCache;
use anyhow::Result;
use async_trait::async_trait;
use std::sync::Arc;
use std::time::Duration;
use tracing::{info, warn};

/// Cache defines the distributed caching contract for orders.
#[async_trait]
pub trait Cache: Send + Sync {
    async fn get(&self, id: &str) -> Result<Option<Order>>;
    async fn set(&self, order: &Order) -> Result<()>;
    async fn get_list(&self, page: i64, limit: i64) -> Result<Option<PaginatedOrders>>;
    async fn set_list(&self, page: i64, limit: i64, list: &PaginatedOrders) -> Result<()>;
    async fn invalidate_lists(&self) -> Result<()>;
}

pub struct RedisOrderCache {
    redis: Option<RedisCache>,
    ttl: Duration,
}

pub fn new_cache(redis: Option<RedisCache>, ttl: Duration) -> Arc<dyn Cache> {
    Arc::new(RedisOrderCache { redis, ttl })
}

pub fn cache_key(id: &str) -> String {
    format!("order:{}", id)
}

pub fn list_cache_key(page: i64, limit: i64) -> String {
    format!("orders:list:page:{}:limit:{}", page, limit)
}

#[async_trait]
impl Cache for RedisOrderCache {
    async fn get(&self, id: &str) -> Result<Option<Order>> {
        let Some(ref redis) = self.redis else {
            return Ok(None);
        };

        match redis.get(&cache_key(id)).await {
            Ok(Some(bytes)) => {
                let order: Order = serde_json::from_slice(&bytes)?;
                info!(id = %id, "⚡ [Redis Cache HIT - Order]");
                Ok(Some(order))
            }
            Ok(None) => Ok(None),
            Err(err) => {
                warn!(id = %id, error = %err, "Redis order cache get failed");
                Err(err)
            }
        }
    }

    async fn set(&self, order: &Order) -> Result<()> {
        let Some(ref redis) = self.redis else {
            return Ok(());
        };

        let data = serde_json::to_vec(order)?;
        let key = cache_key(&order.id.to_string());
        if let Err(err) = redis.set(&key, &data, self.ttl).await {
            warn!(id = %order.id, error = %err, "Redis order cache set failed");
            return Err(err);
        }
        Ok(())
    }

    async fn get_list(&self, page: i64, limit: i64) -> Result<Option<PaginatedOrders>> {
        let Some(ref redis) = self.redis else {
            return Ok(None);
        };

        let key = list_cache_key(page, limit);
        match redis.get(&key).await {
            Ok(Some(bytes)) => {
                let list: PaginatedOrders = serde_json::from_slice(&bytes)?;
                info!(page = page, limit = limit, "⚡ [Redis Cache HIT - Order List]");
                Ok(Some(list))
            }
            Ok(None) => Ok(None),
            Err(err) => {
                warn!(key = %key, error = %err, "Redis order cache get list failed");
                Err(err)
            }
        }
    }

    async fn set_list(&self, page: i64, limit: i64, list: &PaginatedOrders) -> Result<()> {
        let Some(ref redis) = self.redis else {
            return Ok(());
        };

        let data = serde_json::to_vec(list)?;
        let key = list_cache_key(page, limit);
        if let Err(err) = redis.set(&key, &data, self.ttl).await {
            warn!(key = %key, error = %err, "Redis order cache set list failed");
            return Err(err);
        }
        Ok(())
    }

    async fn invalidate_lists(&self) -> Result<()> {
        let Some(ref redis) = self.redis else {
            return Ok(());
        };

        if let Err(err) = redis.del_pattern("orders:list:*").await {
            warn!(error = %err, "Redis order cache list invalidation failed");
            return Err(err);
        }
        Ok(())
    }
}
