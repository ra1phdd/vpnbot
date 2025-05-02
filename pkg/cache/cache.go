package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"nsvpn/pkg/logger"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	log    *logger.Logger
	client *redis.Client
}

func New(log *logger.Logger, client *redis.Client) *Cache {
	return &Cache{log: log, client: client}
}

func (c *Cache) Get(key string, dest any) error {
	data, err := c.client.Get(context.Background(), key).Result()
	if err != nil {
		if data != "" {
			c.log.Error("Failed to get redis data", err, slog.String("key", key), slog.Any("dest", dest))
		}
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

func (c *Cache) Set(key string, value any, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		c.log.Error("Failed to marshal data to json", err, slog.String("key", key), slog.Any("value", value), slog.Duration("ttl", ttl))
		return
	}
	c.client.Set(context.Background(), key, data, ttl)
}

func (c *Cache) Delete(keys ...string) {
	if len(keys) == 0 {
		return
	}

	err := c.client.Del(context.Background(), keys...).Err()
	if err != nil {
		c.log.Error("Failed to delete keys from cache", err, slog.Any("keys", keys))
		return
	}
}
