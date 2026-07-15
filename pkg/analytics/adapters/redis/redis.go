package redis

import (
	"context"
	"sync/atomic"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	goredis "github.com/redis/go-redis/v9"
)

// Ensure Tracker implements analytics.Tracker.
var _ analytics.Tracker = (*Tracker)(nil)

const defaultKeyPrefix = "analytics:hll:"

// Tracker implements analytics.Tracker using Redis HyperLogLog commands
// (PFADD, PFCOUNT, PFMERGE, DEL).
//
// analytics.Config.Precision is ignored — Redis HyperLogLog uses a fixed
// internal representation (~12KB per key).
type Tracker struct {
	client    goredis.Cmdable
	closer    func() error
	keyPrefix string
	closed    atomic.Bool
}

// Option configures a Redis Tracker.
type Option func(*Tracker)

// WithKeyPrefix sets the Redis key prefix (default "analytics:hll:").
func WithKeyPrefix(prefix string) Option {
	return func(t *Tracker) {
		if prefix != "" {
			t.keyPrefix = prefix
		}
	}
}

// New creates a Redis-backed tracker that owns client and closes it on Close.
func New(client *goredis.Client, opts ...Option) analytics.Tracker {
	t := &Tracker{
		client:    client,
		closer:    client.Close,
		keyPrefix: defaultKeyPrefix,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// NewWithCmdable creates a tracker from a shared Cmdable (Close is a no-op
// for the underlying connection).
func NewWithCmdable(client goredis.Cmdable, opts ...Option) analytics.Tracker {
	t := &Tracker{
		client:    client,
		keyPrefix: defaultKeyPrefix,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *Tracker) key(counter string) string {
	return t.keyPrefix + counter
}

func (t *Tracker) guard() error {
	if t.closed.Load() {
		return analytics.ErrClosed
	}
	return nil
}

// Add records an element via PFADD.
func (t *Tracker) Add(ctx context.Context, counter string, element string) error {
	if err := t.guard(); err != nil {
		return err
	}
	if err := t.client.PFAdd(ctx, t.key(counter), element).Err(); err != nil {
		return errors.Wrap(err, "analytics redis PFADD")
	}
	return nil
}

// Count returns PFCOUNT. Missing keys return (0, nil).
func (t *Tracker) Count(ctx context.Context, counter string) (uint64, error) {
	if err := t.guard(); err != nil {
		return 0, err
	}
	n, err := t.client.PFCount(ctx, t.key(counter)).Result()
	if err != nil {
		return 0, errors.Wrap(err, "analytics redis PFCOUNT")
	}
	if n < 0 {
		return 0, nil
	}
	return uint64(n), nil
}

// Reset deletes the counter key. Missing keys are a no-op.
func (t *Tracker) Reset(ctx context.Context, counter string) error {
	if err := t.guard(); err != nil {
		return err
	}
	if err := t.client.Del(ctx, t.key(counter)).Err(); err != nil {
		return errors.Wrap(err, "analytics redis DEL")
	}
	return nil
}

// Merge runs PFMERGE dest source. Returns ErrCounterNotFound if source is missing.
func (t *Tracker) Merge(ctx context.Context, dest, source string) error {
	if err := t.guard(); err != nil {
		return err
	}

	srcKey := t.key(source)
	n, err := t.client.Exists(ctx, srcKey).Result()
	if err != nil {
		return errors.Wrap(err, "analytics redis EXISTS")
	}
	if n == 0 {
		return analytics.ErrCounterNotFound
	}

	if err := t.client.PFMerge(ctx, t.key(dest), srcKey).Err(); err != nil {
		return errors.Wrap(err, "analytics redis PFMERGE")
	}
	return nil
}

// Close marks the tracker closed and closes the owned Redis client when present.
func (t *Tracker) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	if t.closer != nil {
		return t.closer()
	}
	return nil
}
