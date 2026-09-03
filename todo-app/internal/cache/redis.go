package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func Ping(ctx context.Context, c *redis.Client) error {
	if err := c.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}
