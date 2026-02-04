package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	// Best-effort ping: auth service must start even if Redis is down
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("redis ping failed, continuing without redis", "error", err)
	}

	return client, nil
}
