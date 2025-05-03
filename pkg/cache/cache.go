package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"nsvpn/pkg/logger"
	"strings"
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

	err = c.client.Set(context.Background(), key, data, ttl).Err()
	if err != nil {
		c.log.Error("Failed to set key", err, slog.String("key", key), slog.Duration("ttl", ttl))
	}
}

func (c *Cache) Delete(keys ...string) {
	if len(keys) == 0 {
		return
	}

	var keysToDelete []string
	for _, pattern := range keys {
		if strings.ContainsAny(pattern, "*?[\\") {
			iter := c.client.Scan(context.Background(), 0, pattern, 0).Iterator()
			for iter.Next(context.Background()) {
				keysToDelete = append(keysToDelete, iter.Val())
			}
			if err := iter.Err(); err != nil {
				c.log.Error("SCAN error", err, slog.String("pattern", pattern))
			}
		} else {
			keysToDelete = append(keysToDelete, pattern)
		}
	}

	if len(keysToDelete) > 0 {
		err := c.client.Del(context.Background(), keysToDelete...).Err()
		if err != nil {
			c.log.Error("Failed to delete keys", err, slog.Any("keys", keysToDelete))
		}
	}
}
