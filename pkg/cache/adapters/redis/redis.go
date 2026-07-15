package redis

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"reflect"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/redis/go-redis/v9"
)

func init() {
	cache.RegisterDriver("redis", New)
}

// RedisCache implements cache.Cache on Redis (standalone or cluster).
type RedisCache struct {
	client redis.UniversalClient
}

var (
	_ cache.Cache         = (*RedisCache)(nil)
	_ cache.PrefixDeleter = (*RedisCache)(nil)
)

// New connects to Redis using cfg and returns a Cache.
// When cfg.Cluster is true (or cfg.Addrs is non-empty), a Cluster client is used.
func New(cfg cache.Config) (cache.Cache, error) {
	if cfg.Cluster || len(cfg.Addrs) > 0 {
		return newCluster(cfg)
	}

	opts := &redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	if cfg.DialTimeout > 0 {
		opts.DialTimeout = cfg.DialTimeout
	}
	if cfg.ReadTimeout > 0 {
		opts.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		opts.WriteTimeout = cfg.WriteTimeout
	}
	if cfg.TLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, errors.Wrap(err, "failed to connect to redis cache")
	}
	return NewWithClient(client), nil
}

func newCluster(cfg cache.Config) (cache.Cache, error) {
	addrs := append([]string(nil), cfg.Addrs...)
	if len(addrs) == 0 {
		if cfg.Host == "" || cfg.Port == "" {
			return nil, errors.InvalidArgument("redis cluster requires Addrs or Host:Port", nil)
		}
		addrs = []string{cfg.Host + ":" + cfg.Port}
	}

	opts := &redis.ClusterOptions{
		Addrs:    addrs,
		Password: cfg.Password,
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}
	if cfg.DialTimeout > 0 {
		opts.DialTimeout = cfg.DialTimeout
	}
	if cfg.ReadTimeout > 0 {
		opts.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		opts.WriteTimeout = cfg.WriteTimeout
	}
	if cfg.TLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client := redis.NewClusterClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, errors.Wrap(err, "failed to connect to redis cluster")
	}
	return NewWithUniversalClient(client), nil
}

// NewWithClient wraps an existing go-redis client (useful for miniredis tests).
func NewWithClient(client *redis.Client) cache.Cache {
	return &RedisCache{client: client}
}

// NewWithUniversalClient wraps a UniversalClient (standalone or cluster).
func NewWithUniversalClient(client redis.UniversalClient) cache.Cache {
	return &RedisCache{client: client}
}

// NewCluster is a convenience constructor for Redis Cluster from seed addrs.
func NewCluster(addrs []string, password string) (cache.Cache, error) {
	return New(cache.Config{
		Cluster:  true,
		Addrs:    addrs,
		Password: password,
	})
}

func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return cache.ErrKeyNotFound
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

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.Wrap(err, "failed to exists in redis")
	}
	return n > 0, nil
}

func (r *RedisCache) MGet(ctx context.Context, keys []string, dest interface{}) error {
	rv, err := mapDest(dest)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return errors.Wrap(err, "failed to mget from redis")
	}

	elemType := rv.Type().Elem()
	for i, raw := range vals {
		if raw == nil {
			continue
		}
		var data []byte
		switch v := raw.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			continue
		}
		ptr := reflect.New(elemType)
		if err := json.Unmarshal(data, ptr.Interface()); err != nil {
			return errors.Wrap(err, "failed to unmarshal mget value")
		}
		rv.SetMapIndex(reflect.ValueOf(keys[i]), ptr.Elem())
	}
	return nil
}

func (r *RedisCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	pipe := r.client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return errors.Wrap(err, "failed to marshal value")
		}
		pipe.Set(ctx, key, data, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return errors.Wrap(err, "failed to mset to redis")
	}
	return nil
}

func (r *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return errors.Wrap(err, "failed to check key for expire")
	}
	if exists == 0 {
		return cache.ErrKeyNotFound
	}
	if ttl <= 0 {
		if err := r.client.Persist(ctx, key).Err(); err != nil {
			return errors.Wrap(err, "failed to persist key")
		}
		return nil
	}
	ok, err := r.client.Expire(ctx, key, ttl).Result()
	if err != nil {
		return errors.Wrap(err, "failed to expire key")
	}
	if !ok {
		return cache.ErrKeyNotFound
	}
	return nil
}

func (r *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get ttl")
	}
	// go-redis maps Redis TTL: -2s = missing, -1s = no expiry.
	switch {
	case d == -2*time.Second:
		return 0, cache.ErrKeyNotFound
	case d < 0:
		return -1, nil
	default:
		return d, nil
	}
}

func (r *RedisCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(ctx, key, delta).Result()
}

func (r *RedisCache) DeletePrefix(ctx context.Context, prefix string) (int64, error) {
	var cursor uint64
	var deleted int64
	pattern := prefix + "*"
	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return deleted, errors.Wrap(err, "failed to scan keys for prefix delete")
		}
		if len(keys) > 0 {
			n, err := r.client.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, errors.Wrap(err, "failed to delete prefix keys")
			}
			deleted += n
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}

func mapDest(dest interface{}) (reflect.Value, error) {
	if dest == nil {
		return reflect.Value{}, errors.InvalidArgument("mget dest is nil", nil)
	}
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return reflect.Value{}, errors.InvalidArgument("mget dest must be a non-nil pointer to map[string]T", nil)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return reflect.Value{}, errors.InvalidArgument("mget dest must be *map[string]T", nil)
	}
	if rv.IsNil() {
		rv.Set(reflect.MakeMap(rv.Type()))
	}
	return rv, nil
}
