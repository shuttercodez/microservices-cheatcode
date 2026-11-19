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

// Cache defines the distributed caching contract for products.
type Cache interface {
	Get(ctx context.Context, id string) (*Product, error)
	Set(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
	GetList(ctx context.Context, page, limit int) (*PaginatedProducts, error)
	SetList(ctx context.Context, page, limit int, list *PaginatedProducts) error
	InvalidateLists(ctx context.Context) error
}

type redisProductCache struct {
	redis *cache.RedisCache
	ttl   time.Duration
}

// NewCache creates a Redis-backed Product Cache with a configurable TTL.
func NewCache(redis *cache.RedisCache, ttl time.Duration) Cache {
	return &redisProductCache{
		redis: redis,
		ttl:   ttl,
	}
}

func cacheKey(id string) string {
	return fmt.Sprintf("product:%s", id)
}

func listCacheKey(page, limit int) string {
	return fmt.Sprintf("products:list:page:%d:limit:%d", page, limit)
}

// Get retrieves a cached product by ID. Returns (nil, nil) on a cache miss.
func (c *redisProductCache) Get(ctx context.Context, id string) (*Product, error) {
	if c.redis == nil || c.redis.Client == nil {
		return nil, nil
	}

	data, err := c.redis.Client.Get(ctx, cacheKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
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

// Set stores a product in Redis with the configured TTL.
func (c *redisProductCache) Set(ctx context.Context, product *Product) error {
	if c.redis == nil || c.redis.Client == nil || product == nil {
		return nil
	}

	data, err := json.Marshal(product)
	if err != nil {
		return err
	}

	if err := c.redis.Client.Set(ctx, cacheKey(product.ID.String()), data, c.ttl).Err(); err != nil {
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

// GetList retrieves a cached paginated product list. Returns (nil, nil) on a cache miss.
func (c *redisProductCache) GetList(ctx context.Context, page, limit int) (*PaginatedProducts, error) {
	if c.redis == nil || c.redis.Client == nil {
		return nil, nil
	}

	key := listCacheKey(page, limit)
	data, err := c.redis.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		slog.Warn("Redis cache get list failed", "key", key, "error", err)
		return nil, err
	}

	var list PaginatedProducts
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	slog.Info("⚡ [Redis Cache HIT - Product List]", "page", page, "limit", limit)
	return &list, nil
}

// SetList caches a paginated product list in Redis.
func (c *redisProductCache) SetList(ctx context.Context, page, limit int, list *PaginatedProducts) error {
	if c.redis == nil || c.redis.Client == nil || list == nil {
		return nil
	}

	data, err := json.Marshal(list)
	if err != nil {
		return err
	}

	key := listCacheKey(page, limit)
	if err := c.redis.Client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		slog.Warn("Redis cache set list failed", "key", key, "error", err)
		return err
	}
	return nil
}

// InvalidateLists removes all cached product list pages when products change.
func (c *redisProductCache) InvalidateLists(ctx context.Context) error {
	if c.redis == nil || c.redis.Client == nil {
		return nil
	}

	keys, err := c.redis.Client.Keys(ctx, "products:list:*").Result()
	if err != nil {
		slog.Warn("Redis cache keys scan failed", "error", err)
		return err
	}
	if len(keys) > 0 {
		if err := c.redis.Client.Del(ctx, keys...).Err(); err != nil {
			slog.Warn("Redis cache list invalidation failed", "error", err)
			return err
		}
	}
	return nil
}
