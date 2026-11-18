use anyhow::{anyhow, Result};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use std::time::Duration;

/// RedisCache wraps the official `redis::aio::ConnectionManager` for multiplexed distributed caching operations.
#[derive(Clone)]
pub struct RedisCache {
    pub manager: ConnectionManager,
}

impl RedisCache {
    /// Initializes a new Redis ConnectionManager and verifies connectivity with a 3-second timeout.
    pub async fn new(addr: &str) -> Result<Self> {
        let redis_url = if addr.starts_with("redis://") || addr.starts_with("rediss://") {
            addr.to_string()
        } else {
            format!("redis://{}", addr)
        };

        let client = redis::Client::open(redis_url)?;
        let manager = tokio::time::timeout(
            Duration::from_secs(3),
            ConnectionManager::new(client),
        )
        .await
        .map_err(|_| anyhow!("Redis connection timed out"))??;

        let cache = Self { manager };
        cache.ping().await?;
        Ok(cache)
    }

    /// Verifies Redis connectivity for readiness probes.
    pub async fn ping(&self) -> Result<()> {
        let mut conn = self.manager.clone();
        let pong: String = redis::cmd("PING").query_async(&mut conn).await?;
        if pong == "PONG" {
            Ok(())
        } else {
            Err(anyhow!("Unexpected Redis PING response: {}", pong))
        }
    }

    /// Retrieves a cached byte payload by key. Returns `Ok(None)` on a cache miss (`redis::Nil`).
    pub async fn get(&self, key: &str) -> Result<Option<Vec<u8>>> {
        let mut conn = self.manager.clone();
        let data: Option<Vec<u8>> = conn.get(key).await?;
        Ok(data)
    }

    /// Stores a byte payload in Redis with a TTL (`SET_EX`).
    pub async fn set(&self, key: &str, value: &[u8], ttl: Duration) -> Result<()> {
        let mut conn = self.manager.clone();
        let ttl_secs = ttl.as_secs().max(1);
        let _: () = conn.set_ex(key, value, ttl_secs).await?;
        Ok(())
    }

    /// Deletes a key from Redis (`DEL`).
    pub async fn del(&self, key: &str) -> Result<()> {
        let mut conn = self.manager.clone();
        let _: () = conn.del(key).await?;
        Ok(())
    }
}
