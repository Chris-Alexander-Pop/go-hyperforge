package metering

import (
	"context"
	"time"
)

// Meter defines the interface for recording usage events.
type Meter interface {
	// RecordUsage ingests a usage event.
	// Returns ErrInvalidUsage when the event is malformed.
	RecordUsage(ctx context.Context, event UsageEvent) error

	// GetUsage retrieves usage events matching the filter.
	GetUsage(ctx context.Context, filter UsageFilter) ([]UsageEvent, error)

	// Close releases resources held by the meter.
	// The meter should not be used after calling Close.
	Close() error
}

// Rater defines the interface for calculating costs and managing rate cards.
type Rater interface {
	// GetRate returns the price for a specific resource type.
	// Returns ErrRateNotFound when no rate card exists.
	GetRate(ctx context.Context, resourceType string) (*RateCard, error)

	// SetRate creates or updates the rate card for a resource type.
	// Returns ErrInvalidUsage when the rate card is malformed.
	SetRate(ctx context.Context, rate RateCard) error

	// ListRates returns all configured rate cards.
	ListRates(ctx context.Context) ([]RateCard, error)

	// CalculateCost estimates the cost for a given usage.
	CalculateCost(ctx context.Context, usage UsageEvent) (float64, error)

	// Close releases resources held by the rater.
	// The rater should not be used after calling Close.
	Close() error
}

// UsageEvent represents a single consumption record.
type UsageEvent struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	ResourceType string            `json:"resource_type"` // e.g. "compute.instance.small", "storage.standard"
	ResourceID   string            `json:"resource_id"`
	Quantity     float64           `json:"quantity"` // e.g. hours, GB-months
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// UsageFilter defines criteria for querying usage.
type UsageFilter struct {
	TenantID     string    `json:"tenant_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	StartTime    time.Time `json:"start_time,omitempty"`
	EndTime      time.Time `json:"end_time,omitempty"`
}

// RateCard defines the pricing for a resource.
type RateCard struct {
	ResourceType string  `json:"resource_type"`
	PricePerUnit float64 `json:"price_per_unit"`
	Currency     string  `json:"currency"` // e.g. "USD"
	Unit         string  `json:"unit"`     // e.g. "hour", "gb-month"
}

// Config holds configuration for the Metering service.
type Config struct {
	// Driver selects the metering backend.
	// Currently only "memory" is implemented (see adapters/memory).
	// Values such as "prometheus" or "postgres" are reserved for future adapters.
	Driver string `env:"METERING_DRIVER" env-default:"memory"`
}

// ValidateUsageEvent returns ErrInvalidUsage when the event cannot be recorded.
func ValidateUsageEvent(event UsageEvent) error {
	if event.TenantID == "" {
		return ErrInvalidUsage
	}
	if event.ResourceType == "" {
		return ErrInvalidUsage
	}
	if event.Quantity <= 0 {
		return ErrInvalidUsage
	}
	return nil
}

// ValidateRateCard returns ErrInvalidUsage when the rate card is malformed.
func ValidateRateCard(rate RateCard) error {
	if rate.ResourceType == "" {
		return ErrInvalidUsage
	}
	if rate.PricePerUnit < 0 {
		return ErrInvalidUsage
	}
	if rate.Currency == "" {
		return ErrInvalidUsage
	}
	if rate.Unit == "" {
		return ErrInvalidUsage
	}
	return nil
}
