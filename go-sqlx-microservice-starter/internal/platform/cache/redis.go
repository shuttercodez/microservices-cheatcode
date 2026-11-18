package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache wraps the go-redis client for distributed caching operations.
type RedisCache struct {
	Client *redis.Client
}

// NewRedisCache initializes a new Redis client and verifies connectivity.
func NewRedisCache(addr string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &RedisCache{Client: client}, nil
}

// Ping verifies Redis connectivity for readiness probes.
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close safely closes the Redis client connection.
func (r *RedisCache) Close() {
	if r.Client != nil {
		_ = r.Client.Close()
	}
}
