package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func New(cfg cache.Config) (cache.Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Check connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to redis cache")
	}

	return &RedisCache{client: client}, nil
}

func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return errors.New(errors.CodeNotFound, "key not found", nil)
	}
	if err != nil {
		return errors.Wrap(err, "failed to get from redis")
	}

	return json.Unmarshal(val, dest)
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal value")
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return errors.Wrap(err, "failed to set to redis")
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(ctx, key, delta).Result()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
