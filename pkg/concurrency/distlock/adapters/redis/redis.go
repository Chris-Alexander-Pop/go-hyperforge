package redis

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency/distlock"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Adapter implements distlock.Locker using Redis.
type Adapter struct {
	client redis.Cmdable
	prefix string
}

func New(client redis.Cmdable, prefix string) *Adapter {
	if prefix == "" {
		prefix = "lock:"
	}
	return &Adapter{
		client: client,
		prefix: prefix,
	}
}

func (a *Adapter) NewLock(key string, ttl time.Duration) distlock.Lock {
	return &Lock{
		client: a.client,
		key:    a.prefix + key,
		value:  uuid.New().String(), // Unique identifier for this lock holder
		ttl:    ttl,
	}
}

func (a *Adapter) Close() error {
	return nil
}

// Lock implements a Redis-based lock.
type Lock struct {
	client redis.Cmdable
	key    string
	value  string
	ttl    time.Duration
	held   bool
}

// Acquire attempts to acquire the lock using SET NX (set if not exists).
func (l *Lock) Acquire(ctx context.Context) (bool, error) {
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

func (l *Lock) Release(ctx context.Context) error {
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

func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
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

func (l *Lock) IsHeld() bool {
	return l.held
}
