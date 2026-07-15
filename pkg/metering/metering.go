package metering

import (
	"context"
	"sort"
	"time"
)

// Meter defines the interface for recording usage events.
type Meter interface {
	// RecordUsage ingests a usage event.
	// Returns ErrInvalidUsage when the event is malformed.
	RecordUsage(ctx context.Context, event UsageEvent) error

	// GetUsage retrieves usage events matching the filter.
	GetUsage(ctx context.Context, filter UsageFilter) ([]UsageEvent, error)

	// PeriodAggregate buckets matching usage into fixed-width periods.
	// period must be > 0. Buckets are ordered by PeriodStart ascending.
	PeriodAggregate(ctx context.Context, filter UsageFilter, period time.Duration) ([]PeriodBucket, error)

	// SummarizeUsage returns totals for matching usage (by resource type and overall).
	SummarizeUsage(ctx context.Context, filter UsageFilter) (*UsageSummary, error)

	// Close releases resources held by the meter.
	// The meter should not be used after calling Close.
	Close() error
}

// PeriodBucket is a time-bucketed usage aggregate.
type PeriodBucket struct {
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	TenantID     string    `json:"tenant_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	Quantity     float64   `json:"quantity"`
	EventCount   int       `json:"event_count"`
}

// UsageSummary aggregates usage across a filter window.
type UsageSummary struct {
	TenantID       string             `json:"tenant_id,omitempty"`
	TotalQuantity  float64            `json:"total_quantity"`
	EventCount     int                `json:"event_count"`
	ByResourceType map[string]float64 `json:"by_resource_type"`
	StartTime      time.Time          `json:"start_time,omitempty"`
	EndTime        time.Time          `json:"end_time,omitempty"`
}

// SummarizeEvents builds a UsageSummary from already-fetched events.
func SummarizeEvents(events []UsageEvent, filter UsageFilter) *UsageSummary {
	sum := &UsageSummary{
		TenantID:       filter.TenantID,
		ByResourceType: make(map[string]float64),
		StartTime:      filter.StartTime,
		EndTime:        filter.EndTime,
	}
	for _, e := range events {
		sum.TotalQuantity += e.Quantity
		sum.EventCount++
		sum.ByResourceType[e.ResourceType] += e.Quantity
	}
	return sum
}

// BucketEvents groups events into fixed-width period buckets keyed by
// (periodStart, resourceType). Events with zero Timestamp are skipped.
func BucketEvents(events []UsageEvent, period time.Duration) []PeriodBucket {
	if period <= 0 {
		return nil
	}
	type key struct {
		start int64
		res   string
		ten   string
	}
	acc := make(map[key]*PeriodBucket)
	for _, e := range events {
		if e.Timestamp.IsZero() {
			continue
		}
		ts := e.Timestamp.UTC()
		startUnix := ts.UnixNano() - (ts.UnixNano() % period.Nanoseconds())
		start := time.Unix(0, startUnix).UTC()
		k := key{start: startUnix, res: e.ResourceType, ten: e.TenantID}
		b, ok := acc[k]
		if !ok {
			b = &PeriodBucket{
				PeriodStart:  start,
				PeriodEnd:    start.Add(period),
				TenantID:     e.TenantID,
				ResourceType: e.ResourceType,
			}
			acc[k] = b
		}
		b.Quantity += e.Quantity
		b.EventCount++
	}
	out := make([]PeriodBucket, 0, len(acc))
	for _, b := range acc {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PeriodStart.Equal(out[j].PeriodStart) {
			return out[i].ResourceType < out[j].ResourceType
		}
		return out[i].PeriodStart.Before(out[j].PeriodStart)
	})
	return out
}

// DefaultPeriodAggregate implements PeriodAggregate via GetUsage + BucketEvents.
func DefaultPeriodAggregate(ctx context.Context, m Meter, filter UsageFilter, period time.Duration) ([]PeriodBucket, error) {
	if period <= 0 {
		return nil, ErrInvalidUsage
	}
	events, err := m.GetUsage(ctx, filter)
	if err != nil {
		return nil, err
	}
	return BucketEvents(events, period), nil
}

// DefaultSummarizeUsage implements SummarizeUsage via GetUsage + SummarizeEvents.
func DefaultSummarizeUsage(ctx context.Context, m Meter, filter UsageFilter) (*UsageSummary, error) {
	events, err := m.GetUsage(ctx, filter)
	if err != nil {
		return nil, err
	}
	return SummarizeEvents(events, filter), nil
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
	// Driver selects the metering backend: "memory", "prometheus", "postgres".
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
