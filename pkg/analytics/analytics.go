package analytics

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

// Config holds configuration for an Analytics tracker.
type Config struct {
	// Precision for HyperLogLog sketches (4–16 inclusive). Default 14.
	// Aligned with pkg/datastructures/hyperloglog.
	// Redis adapters ignore this field (Redis HyperLogLog uses a fixed structure).
	Precision uint8 `env:"ANALYTICS_PRECISION" env-default:"14" validate:"gte=4,lte=16"`
}

// DefaultConfig returns Config with package defaults applied.
func DefaultConfig() Config {
	return Config{Precision: 14}
}

// Normalize applies defaults (Precision 0 → 14) then validates with pkg/validator.
func (c Config) Normalize() (Config, error) {
	if c.Precision == 0 {
		c.Precision = 14
	}
	if err := validator.New().ValidateStruct(context.Background(), c); err != nil {
		// ValidateStruct already returns errors.InvalidArgument on tag failures.
		if errors.IsCode(err, errors.CodeInvalidArgument) {
			return Config{}, err
		}
		return Config{}, errors.InvalidArgument("invalid analytics config", err)
	}
	return c, nil
}

// Tracker estimates unique element counts via HyperLogLog sketches.
//
// Scope is uniqueness / cardinality only — not event warehouses, funnels,
// sessionization, or OLAP analytics (see package doc).
type Tracker interface {
	// Add records an element for the given counter name.
	// Creates the counter if it does not already exist.
	Add(ctx context.Context, counter string, element string) error

	// Count returns the estimated unique count for the given counter.
	// Missing counters return (0, nil) — they do not return ErrCounterNotFound.
	Count(ctx context.Context, counter string) (uint64, error)

	// Reset clears a specific counter. Missing counters are a no-op (nil error).
	Reset(ctx context.Context, counter string) error

	// Merge merges the HyperLogLog sketch of source into dest (union cardinality).
	// Returns ErrCounterNotFound if source does not exist.
	// Creates dest if it does not already exist.
	Merge(ctx context.Context, dest, source string) error

	// Close releases resources held by the tracker.
	// After Close, further operations return ErrClosed.
	Close() error
}
