package cache

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// PrefixDeleter is implemented by backends that can delete by key prefix.
type PrefixDeleter interface {
	DeletePrefix(ctx context.Context, prefix string) (int64, error)
}

// InvalidatePrefix deletes all keys with the given prefix.
// Unwraps Instrumented/Resilient/Bloom wrappers, then requires PrefixDeleter
// (memory and redis implement it). Returns the number of keys deleted.
func InvalidatePrefix(ctx context.Context, c Cache, prefix string) (int64, error) {
	if c == nil {
		return 0, errors.InvalidArgument("cache is nil", nil)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	for {
		u, ok := c.(interface{ Unwrap() Cache })
		if !ok {
			break
		}
		next := u.Unwrap()
		if next == nil || next == c {
			break
		}
		c = next
	}

	if pd, ok := c.(PrefixDeleter); ok {
		return pd.DeletePrefix(ctx, prefix)
	}
	return 0, errors.Unimplemented("cache backend does not support prefix invalidation", nil)
}
