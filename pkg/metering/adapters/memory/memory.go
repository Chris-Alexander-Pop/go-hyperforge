package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/metering"
	"github.com/google/uuid"
)

// MemoryMetering implements both Meter and Rater interfaces in-memory.
type MemoryMetering struct {
	usage []metering.UsageEvent
	rates map[string]metering.RateCard
	mu    *concurrency.SmartRWMutex
}

// New creates a new MemoryMetering adapter.
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
	m.mu.Lock()
	defer m.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	m.usage = append(m.usage, event)
	return nil
}

func (m *MemoryMetering) GetUsage(ctx context.Context, filter metering.UsageFilter) ([]metering.UsageEvent, error) {
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

func (m *MemoryMetering) GetRate(ctx context.Context, resourceType string) (*metering.RateCard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rate, ok := m.rates[resourceType]
	if !ok {
		return nil, metering.ErrRateNotFound
	}
	return &rate, nil
}

func (m *MemoryMetering) CalculateCost(ctx context.Context, usage metering.UsageEvent) (float64, error) {
	rate, err := m.GetRate(ctx, usage.ResourceType)
	if err != nil {
		return 0, err
	}

	return usage.Quantity * rate.PricePerUnit, nil
}
