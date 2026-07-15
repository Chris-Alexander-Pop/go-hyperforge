// Package prometheus exports metering.Meter usage as Prometheus text exposition.
package prometheus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	"github.com/google/uuid"
)

// Ensure compile-time compliance.
var _ metering.Meter = (*Exporter)(nil)

// Exporter is a Meter that retains events for GetUsage and exposes Prometheus metrics.
type Exporter struct {
	mu     *concurrency.SmartRWMutex
	usage  []metering.UsageEvent
	totals map[seriesKey]float64
	closed atomic.Bool
}

type seriesKey struct {
	TenantID     string
	ResourceType string
}

// New creates a Prometheus metering exporter.
func New() *Exporter {
	return &Exporter{
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "metering-prometheus"}),
		usage:  make([]metering.UsageEvent, 0),
		totals: make(map[seriesKey]float64),
	}
}

// RecordUsage stores the event and increments the exported counter.
func (e *Exporter) RecordUsage(ctx context.Context, event metering.UsageEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if e.closed.Load() {
		return metering.ErrClosed(nil)
	}
	if err := metering.ValidateUsageEvent(event); err != nil {
		return err
	}
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

	e.mu.Lock()
	defer e.mu.Unlock()
	e.usage = append(e.usage, event)
	key := seriesKey{TenantID: event.TenantID, ResourceType: event.ResourceType}
	e.totals[key] += event.Quantity
	return nil
}

// GetUsage returns matching recorded events.
func (e *Exporter) GetUsage(ctx context.Context, filter metering.UsageFilter) ([]metering.UsageEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e.closed.Load() {
		return nil, metering.ErrClosed(nil)
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]metering.UsageEvent, 0)
	for _, ev := range e.usage {
		if filter.TenantID != "" && ev.TenantID != filter.TenantID {
			continue
		}
		if filter.ResourceType != "" && ev.ResourceType != filter.ResourceType {
			continue
		}
		if !filter.StartTime.IsZero() && ev.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && ev.Timestamp.After(filter.EndTime) {
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}

// PeriodAggregate buckets usage into fixed-width periods.
func (e *Exporter) PeriodAggregate(ctx context.Context, filter metering.UsageFilter, period time.Duration) ([]metering.PeriodBucket, error) {
	return metering.DefaultPeriodAggregate(ctx, e, filter, period)
}

// SummarizeUsage returns totals for matching usage.
func (e *Exporter) SummarizeUsage(ctx context.Context, filter metering.UsageFilter) (*metering.UsageSummary, error) {
	return metering.DefaultSummarizeUsage(ctx, e, filter)
}

// Close marks the exporter closed.
func (e *Exporter) Close() error {
	e.closed.Store(true)
	return nil
}

// Gather writes Prometheus text exposition format to w.
func (e *Exporter) Gather(w io.Writer) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keys := make([]seriesKey, 0, len(e.totals))
	for k := range e.totals {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].TenantID == keys[j].TenantID {
			return keys[i].ResourceType < keys[j].ResourceType
		}
		return keys[i].TenantID < keys[j].TenantID
	})

	var buf bytes.Buffer
	buf.WriteString("# HELP metering_usage_quantity_total Cumulative metered quantity by tenant and resource.\n")
	buf.WriteString("# TYPE metering_usage_quantity_total counter\n")
	for _, k := range keys {
		fmt.Fprintf(&buf, "metering_usage_quantity_total{tenant_id=%q,resource_type=%q} %s\n",
			k.TenantID, k.ResourceType, formatFloat(e.totals[k]))
	}
	buf.WriteString("# HELP metering_usage_events_total Number of usage events recorded.\n")
	buf.WriteString("# TYPE metering_usage_events_total counter\n")
	fmt.Fprintf(&buf, "metering_usage_events_total %d\n", len(e.usage))
	_, err := w.Write(buf.Bytes())
	return err
}

// Handler returns an HTTP handler serving /metrics-style exposition.
func (e *Exporter) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_ = e.Gather(w)
	})
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
