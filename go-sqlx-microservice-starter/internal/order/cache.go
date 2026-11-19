package order

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

// Cache defines the distributed caching contract for orders.
type Cache interface {
	Get(ctx context.Context, id string) (*Order, error)
	Set(ctx context.Context, order *Order) error
	GetList(ctx context.Context, page, limit int) (*PaginatedOrders, error)
	SetList(ctx context.Context, page, limit int, list *PaginatedOrders) error
	InvalidateLists(ctx context.Context) error
}

type redisOrderCache struct {
	redis *cache.RedisCache
	ttl   time.Duration
}

// NewCache creates a Redis-backed Order Cache with a configurable TTL.
func NewCache(redis *cache.RedisCache, ttl time.Duration) Cache {
	return &redisOrderCache{
		redis: redis,
		ttl:   ttl,
	}
}

func cacheKey(id string) string {
	return fmt.Sprintf("order:%s", id)
}

func listCacheKey(page, limit int) string {
	return fmt.Sprintf("orders:list:page:%d:limit:%d", page, limit)
}

// Get retrieves a cached order by ID. Returns (nil, nil) on a cache miss.
func (c *redisOrderCache) Get(ctx context.Context, id string) (*Order, error) {
	if c.redis == nil || c.redis.Client == nil {
		return nil, nil
	}

	data, err := c.redis.Client.Get(ctx, cacheKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		slog.Warn("Redis order cache get failed", "id", id, "error", err)
		return nil, err
	}

	var order Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}

	slog.Info("⚡ [Redis Cache HIT - Order]", "id", id)
	return &order, nil
}

// Set stores an order in Redis with the configured TTL.
func (c *redisOrderCache) Set(ctx context.Context, order *Order) error {
	if c.redis == nil || c.redis.Client == nil || order == nil {
		return nil
	}

	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	if err := c.redis.Client.Set(ctx, cacheKey(order.ID.String()), data, c.ttl).Err(); err != nil {
		slog.Warn("Redis order cache set failed", "id", order.ID, "error", err)
		return err
	}
	return nil
}

// GetList retrieves a cached paginated order list. Returns (nil, nil) on a cache miss.
func (c *redisOrderCache) GetList(ctx context.Context, page, limit int) (*PaginatedOrders, error) {
	if c.redis == nil || c.redis.Client == nil {
		return nil, nil
	}

	key := listCacheKey(page, limit)
	data, err := c.redis.Client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		slog.Warn("Redis order cache get list failed", "key", key, "error", err)
		return nil, err
	}

	var list PaginatedOrders
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	slog.Info("⚡ [Redis Cache HIT - Order List]", "page", page, "limit", limit)
	return &list, nil
}

// SetList caches a paginated order list in Redis.
func (c *redisOrderCache) SetList(ctx context.Context, page, limit int, list *PaginatedOrders) error {
	if c.redis == nil || c.redis.Client == nil || list == nil {
		return nil
	}

	data, err := json.Marshal(list)
	if err != nil {
		return err
	}

	key := listCacheKey(page, limit)
	if err := c.redis.Client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		slog.Warn("Redis order cache set list failed", "key", key, "error", err)
		return err
	}
	return nil
}

// InvalidateLists removes all cached order list pages when a new order is created.
func (c *redisOrderCache) InvalidateLists(ctx context.Context) error {
	if c.redis == nil || c.redis.Client == nil {
		return nil
	}

	keys, err := c.redis.Client.Keys(ctx, "orders:list:*").Result()
	if err != nil {
		slog.Warn("Redis order cache keys scan failed", "error", err)
		return err
	}
	if len(keys) > 0 {
		if err := c.redis.Client.Del(ctx, keys...).Err(); err != nil {
			slog.Warn("Redis order cache list invalidation failed", "error", err)
			return err
		}
	}
	return nil
}
