package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis client for cache-aside operations.
// A nil redis client is safe — all operations fall through to the fetch function.
type Client struct {
	redis *redis.Client
}

// NewClient creates a new cache client. Passing a nil redis client is safe;
// all cache operations will transparently delegate to the underlying data source.
func NewClient(redis *redis.Client) *Client {
	return &Client{redis: redis}
}

// Get implements the cache-aside pattern with generics.
// On cache hit, returns the cached value. On miss, calls fetch, caches the result, and returns it.
// Redis errors are logged and cause a fallback to fetch (graceful degradation).
func Get[T any](ctx context.Context, c *Client, key string, ttl time.Duration, fetch func(ctx context.Context) (T, error)) (T, error) {
	if c == nil || c.redis == nil {
		return fetch(ctx)
	}

	data, err := c.redis.Get(ctx, key).Bytes()
	if err == nil {
		var val T
		if err := json.Unmarshal(data, &val); err == nil {
			return val, nil
		} else {
			slog.Warn("cache unmarshal failed, fetching from source", "key", key, "error", err)
		}
	} else if err != redis.Nil {
		slog.Warn("cache get failed, fetching from source", "key", key, "error", err)
	}

	val, fetchErr := fetch(ctx)
	if fetchErr != nil {
		return val, fetchErr
	}

	cacheSet(c, key, val, ttl)

	return val, nil
}

// Delete removes one or more keys from the cache.
// Errors are logged but never returned — stale data expires via TTL.
func Delete(ctx context.Context, c *Client, keys ...string) {
	if c == nil || c.redis == nil || len(keys) == 0 {
		return
	}
	if err := c.redis.Del(ctx, keys...).Err(); err != nil {
		slog.Warn("cache invalidation failed", "keys", keys, "error", err)
	}
}

func cacheSet(c *Client, key string, val any, ttl time.Duration) {
	data, err := json.Marshal(val)
	if err != nil {
		slog.Warn("cache marshal failed", "key", key, "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.redis.Set(ctx, key, data, ttlWithJitter(ttl)).Err(); err != nil {
		slog.Warn("cache set failed", "key", key, "error", err)
	}
}

// ttlWithJitter adds ±15% jitter to prevent cache stampedes.
func ttlWithJitter(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}
	// Range: base * [0.85, 1.15]
	jitter := time.Duration(rand.Int63n(int64(base) * 30 / 100))
	return base - base*15/100 + jitter
}
