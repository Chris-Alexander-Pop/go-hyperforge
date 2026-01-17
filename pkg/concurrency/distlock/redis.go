package distlock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisLocker implements distributed locking using Redis.
type RedisLocker struct {
	client redis.Cmdable
	prefix string
}

// NewRedisLocker creates a new Redis-based distributed locker.
func NewRedisLocker(client redis.Cmdable, prefix string) *RedisLocker {
	if prefix == "" {
		prefix = "lock:"
	}
	return &RedisLocker{
		client: client,
		prefix: prefix,
	}
}

func (l *RedisLocker) NewLock(key string, ttl time.Duration) Lock {
	return &redisLock{
		client: l.client,
		key:    l.prefix + key,
		value:  uuid.New().String(), // Unique identifier for this lock holder
		ttl:    ttl,
	}
}

func (l *RedisLocker) Close() error {
	return nil
}

type redisLock struct {
	client redis.Cmdable
	key    string
	value  string
	ttl    time.Duration
	held   bool
}

// Acquire attempts to acquire the lock using SET NX (set if not exists).
func (l *redisLock) Acquire(ctx context.Context) (bool, error) {
	success, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return false, err
	}
	l.held = success
	return success, nil
}

// Release releases the lock if we still hold it.
// Uses a Lua script to ensure atomicity (only delete if value matches).
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`)

func (l *redisLock) Release(ctx context.Context) error {
	if !l.held {
		return nil
	}

	result, err := releaseScript.Run(ctx, l.client, []string{l.key}, l.value).Int64()
	if err != nil {
		return err
	}

	l.held = result == 1
	return nil
}

// Extend extends the lock's TTL if we still hold it.
var extendScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
`)

func (l *redisLock) Extend(ctx context.Context, ttl time.Duration) error {
	if !l.held {
		return nil
	}

	result, err := extendScript.Run(ctx, l.client, []string{l.key}, l.value, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}

	l.held = result == 1
	return nil
}

func (l *redisLock) IsHeld() bool {
	return l.held
}

// AcquireWithRetry attempts to acquire the lock with retries.
func AcquireWithRetry(ctx context.Context, lock Lock, retryDelay time.Duration, maxRetries int) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		acquired, err := lock.Acquire(ctx)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(retryDelay):
		}
	}
	return false, nil
}
