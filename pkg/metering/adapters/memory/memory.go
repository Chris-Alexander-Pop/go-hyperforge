package memory

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	"github.com/google/uuid"
)

// Ensure MemoryMetering implements Meter and Rater at compile time.
var (
	_ metering.Meter = (*MemoryMetering)(nil)
	_ metering.Rater = (*MemoryMetering)(nil)
)

// MemoryMetering implements both Meter and Rater interfaces in-memory.
type MemoryMetering struct {
	usage  []metering.UsageEvent
	rates  map[string]metering.RateCard
	mu     *concurrency.SmartRWMutex
	closed atomic.Bool
}

// New creates a new MemoryMetering adapter with seeded default rates.
func New() *MemoryMetering {
	m := &MemoryMetering{
		usage: make([]metering.UsageEvent, 0),
		rates: make(map[string]metering.RateCard),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-metering",
		}),
	}

	// Seed some default rates
	m.rates["compute.instance.small"] = metering.RateCard{
		ResourceType: "compute.instance.small",
		PricePerUnit: 0.02,
		Currency:     "USD",
		Unit:         "hour",
	}
	m.rates["storage.standard"] = metering.RateCard{
		ResourceType: "storage.standard",
		PricePerUnit: 0.10,
		Currency:     "USD",
		Unit:         "gb-month",
	}

	return m
}

func (m *MemoryMetering) RecordUsage(ctx context.Context, event metering.UsageEvent) error {
	if err := m.checkClosed(); err != nil {
		return err
	}
	if err := metering.ValidateUsageEvent(event); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Metadata != nil {
		cp := make(map[string]string, len(event.Metadata))
		for k, v := range event.Metadata {
			cp[k] = v
		}
		event.Metadata = cp
	}
	m.usage = append(m.usage, event)
	return nil
}

func (m *MemoryMetering) GetUsage(ctx context.Context, filter metering.UsageFilter) ([]metering.UsageEvent, error) {
	if err := m.checkClosed(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]metering.UsageEvent, 0, len(m.usage))
	for _, e := range m.usage {
		if filter.TenantID != "" && e.TenantID != filter.TenantID {
			continue
		}
		if filter.ResourceType != "" && e.ResourceType != filter.ResourceType {
			continue
		}
		if !filter.StartTime.IsZero() && e.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && e.Timestamp.After(filter.EndTime) {
			continue
		}
		results = append(results, e)
	}
	return results, nil
}

// PeriodAggregate buckets usage into fixed-width periods.
func (m *MemoryMetering) PeriodAggregate(ctx context.Context, filter metering.UsageFilter, period time.Duration) ([]metering.PeriodBucket, error) {
	return metering.DefaultPeriodAggregate(ctx, m, filter, period)
}

// SummarizeUsage returns totals for matching usage.
func (m *MemoryMetering) SummarizeUsage(ctx context.Context, filter metering.UsageFilter) (*metering.UsageSummary, error) {
	return metering.DefaultSummarizeUsage(ctx, m, filter)
}

func (m *MemoryMetering) GetRate(ctx context.Context, resourceType string) (*metering.RateCard, error) {
	if err := m.checkClosed(); err != nil {
		return nil, err
	}
	if resourceType == "" {
		return nil, metering.ErrInvalidUsage
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	rate, ok := m.rates[resourceType]
	if !ok {
		return nil, metering.ErrRateNotFound
	}
	cp := rate
	return &cp, nil
}

func (m *MemoryMetering) SetRate(ctx context.Context, rate metering.RateCard) error {
	if err := m.checkClosed(); err != nil {
		return err
	}
	if err := metering.ValidateRateCard(rate); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.rates[rate.ResourceType] = rate
	return nil
}

func (m *MemoryMetering) ListRates(ctx context.Context) ([]metering.RateCard, error) {
	if err := m.checkClosed(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]metering.RateCard, 0, len(m.rates))
	for _, rate := range m.rates {
		out = append(out, rate)
	}
	return out, nil
}

func (m *MemoryMetering) CalculateCost(ctx context.Context, usage metering.UsageEvent) (float64, error) {
	if err := m.checkClosed(); err != nil {
		return 0, err
	}
	if err := metering.ValidateUsageEvent(usage); err != nil {
		return 0, err
	}

	rate, err := m.GetRate(ctx, usage.ResourceType)
	if err != nil {
		return 0, err
	}

	return usage.Quantity * rate.PricePerUnit, nil
}

// Close marks the adapter closed. Subsequent operations return ErrClosed.
func (m *MemoryMetering) Close() error {
	m.closed.Store(true)
	return nil
}

func (m *MemoryMetering) checkClosed() error {
	if m.closed.Load() {
		return metering.ErrClosed(nil)
	}
	return nil
}
