package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
)

// RedisCache - implementation of Cache interface
type RedisCache struct {
	ctx    context.Context
	client *redis.Client
}

// InitRedisCache - create new instance of RedisCache
// host and port - connection to Redis instance
func InitRedisCache(ctx context.Context, host string, port int) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// check connection by setting test value
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, "redis ping")
	}

	return &RedisCache{
		client: rdb,
	}, nil
}

// Add - add rand value with expiration (in seconds) to cache
func (c *RedisCache) Add(ctx context.Context, key string, expiration int64) error {
	return c.client.Set(ctx, key, "1", time.Duration(expiration)*time.Second).Err()
}

// Exist - check existence of int key in cache
func (c *RedisCache) Exist(ctx context.Context, key string) (bool, error) {
	val, err := c.client.Exists(ctx, key).Result()
	return val == 1, err
}

// Delete - delete key from cache
func (c *RedisCache) Delete(ctx context.Context, key string) {
	c.client.Del(ctx, key)
}
