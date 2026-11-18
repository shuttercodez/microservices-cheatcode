package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go-sqlx-microservice-starter/internal/platform/cache"

	"github.com/redis/go-redis/v9"
)

const productCacheTTL = 5 * time.Minute

// Cache defines the distributed caching contract for products (equivalent to IDistributedCache in .NET).
type Cache interface {
	Get(ctx context.Context, id string) (*Product, error)
	Set(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
}

type redisProductCache struct {
	redis *cache.RedisCache
}

// NewCache creates a Redis-backed Product Cache.
func NewCache(redis *cache.RedisCache) Cache {
	return &redisProductCache{redis: redis}
}

func cacheKey(id string) string {
	return fmt.Sprintf("product:%s", id)
}

// Get retrieves a cached product by ID. Returns (nil, nil) on a cache miss.
func (c *redisProductCache) Get(ctx context.Context, id string) (*Product, error) {
	if c.redis == nil || c.redis.Client == nil {
		return nil, nil
	}

	data, err := c.redis.Client.Get(ctx, cacheKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Cache miss
		}
		slog.Warn("Redis cache get failed", "id", id, "error", err)
		return nil, err
	}

	var product Product
	if err := json.Unmarshal(data, &product); err != nil {
		return nil, err
	}

	slog.Info("⚡ [Redis Cache HIT]", "id", id, "sku", product.SKU)
	return &product, nil
}

// Set stores a product in Redis with a default TTL.
func (c *redisProductCache) Set(ctx context.Context, product *Product) error {
	if c.redis == nil || c.redis.Client == nil || product == nil {
		return nil
	}

	data, err := json.Marshal(product)
	if err != nil {
		return err
	}

	if err := c.redis.Client.Set(ctx, cacheKey(product.ID.String()), data, productCacheTTL).Err(); err != nil {
		slog.Warn("Redis cache set failed", "id", product.ID, "error", err)
		return err
	}
	return nil
}

// Delete invalidates a product entry in Redis when it is updated or deleted.
func (c *redisProductCache) Delete(ctx context.Context, id string) error {
	if c.redis == nil || c.redis.Client == nil {
		return nil
	}

	if err := c.redis.Client.Del(ctx, cacheKey(id)).Err(); err != nil {
		slog.Warn("Redis cache delete failed", "id", id, "error", err)
		return err
	}
	return nil
}
