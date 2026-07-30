package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SemanticCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttlSeconds int) *SemanticCache {
	return &SemanticCache{
		rdb: rdb,
		ttl: time.Duration(ttlSeconds) * time.Second,
	}
}

func cacheKey(messages []map[string]string, model string) string {
	data, _ := json.Marshal(map[string]any{"messages": messages, "model": model})
	hash := sha256.Sum256(data)
	return fmt.Sprintf("nexusllm:cache:%x", hash)
}

func (c *SemanticCache) Get(ctx context.Context, messages []map[string]string, model string) (string, bool) {
	key := cacheKey(messages, model)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func (c *SemanticCache) Set(ctx context.Context, messages []map[string]string, model, response string) error {
	key := cacheKey(messages, model)
	return c.rdb.Set(ctx, key, response, c.ttl).Err()
}

func (c *SemanticCache) Delete(ctx context.Context, messages []map[string]string, model string) error {
	return c.rdb.Del(ctx, cacheKey(messages, model)).Err()
}
