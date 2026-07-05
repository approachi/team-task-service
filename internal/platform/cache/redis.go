// Package cache constructs the shared Redis client. Phase 1 only uses it for
// a startup health check — task-list caching is a later phase (see
// docs/COVER_LETTER.md), but the client is wired up now so that phase only
// needs to add a decorator, not new plumbing.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/zhuk/team-task-service/internal/config"
)

func Connect(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	return client, nil
}
