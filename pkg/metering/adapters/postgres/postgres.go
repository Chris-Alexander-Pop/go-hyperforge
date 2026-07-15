// Package postgres persists metering.Meter (and optionally Rater) via database/sql.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
	"github.com/google/uuid"
)

// Dialect selects SQL placeholder style.
type Dialect int

const (
	DialectSQLite Dialect = iota
	DialectPostgres
)

// Config configures the metering SQL adapter.
type Config struct {
	Dialect Dialect
}

// Store implements metering.Meter and metering.Rater.
type Store struct {
	db      *sql.DB
	dialect Dialect
	closed  atomic.Bool
}

var (
	_ metering.Meter = (*Store)(nil)
	_ metering.Rater = (*Store)(nil)
)

// New wraps an existing *sql.DB. Call Migrate before use.
func New(db *sql.DB, cfg Config) (*Store, error) {
	if db == nil {
		return nil, pkgerrors.InvalidArgument("db is required", nil)
	}
	return &Store{db: db, dialect: cfg.Dialect}, nil
}

func (s *Store) rewrite(query string) string {
	if s.dialect != DialectPostgres {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

// Migrate creates metering tables.
func (s *Store) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS metering_usage (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL DEFAULT '',
			quantity REAL NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			metadata TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metering_usage_tenant ON metering_usage(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_metering_usage_ts ON metering_usage(timestamp)`,
		`CREATE TABLE IF NOT EXISTS metering_rates (
			resource_type TEXT PRIMARY KEY,
			price_per_unit REAL NOT NULL,
			currency TEXT NOT NULL,
			unit TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return pkgerrors.Internal("metering migrate failed", err)
		}
	}
	return nil
}

func (s *Store) checkClosed() error {
	if s.closed.Load() {
		return metering.ErrClosed(nil)
	}
	return nil
}

// RecordUsage inserts a usage event.
func (s *Store) RecordUsage(ctx context.Context, event metering.UsageEvent) error {
	if err := s.checkClosed(); err != nil {
		return err
	}
	if err := metering.ValidateUsageEvent(event); err != nil {
		return err
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	meta := "{}"
	if event.Metadata != nil {
		raw, err := json.Marshal(event.Metadata)
		if err != nil {
			return pkgerrors.Internal("failed to encode metadata", err)
		}
		meta = string(raw)
	}
	_, err := s.db.ExecContext(ctx, s.rewrite(`
INSERT INTO metering_usage (id, tenant_id, resource_type, resource_id, quantity, timestamp, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?)
`), event.ID, event.TenantID, event.ResourceType, event.ResourceID, event.Quantity, event.Timestamp.UTC(), meta)
	if err != nil {
		return pkgerrors.Internal("failed to record usage", err)
	}
	return nil
}

// GetUsage returns matching usage events.
func (s *Store) GetUsage(ctx context.Context, filter metering.UsageFilter) ([]metering.UsageEvent, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, tenant_id, resource_type, resource_id, quantity, timestamp, metadata FROM metering_usage`)
	if err != nil {
		return nil, pkgerrors.Internal("failed to query usage", err)
	}
	defer rows.Close()
	out := make([]metering.UsageEvent, 0)
	for rows.Next() {
		var e metering.UsageEvent
		var meta string
		var ts time.Time
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ResourceType, &e.ResourceID, &e.Quantity, &ts, &meta); err != nil {
			return nil, pkgerrors.Internal("failed to scan usage", err)
		}
		e.Timestamp = ts
		if meta != "" && meta != "{}" {
			_ = json.Unmarshal([]byte(meta), &e.Metadata)
		}
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
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetRate returns a rate card.
func (s *Store) GetRate(ctx context.Context, resourceType string) (*metering.RateCard, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	if resourceType == "" {
		return nil, metering.ErrInvalidUsage
	}
	var rate metering.RateCard
	err := s.db.QueryRowContext(ctx, s.rewrite(`
SELECT resource_type, price_per_unit, currency, unit FROM metering_rates WHERE resource_type = ?
`), resourceType).Scan(&rate.ResourceType, &rate.PricePerUnit, &rate.Currency, &rate.Unit)
	if err == sql.ErrNoRows {
		return nil, metering.ErrRateNotFound
	}
	if err != nil {
		return nil, pkgerrors.Internal("failed to get rate", err)
	}
	return &rate, nil
}

// SetRate upserts a rate card.
func (s *Store) SetRate(ctx context.Context, rate metering.RateCard) error {
	if err := s.checkClosed(); err != nil {
		return err
	}
	if err := metering.ValidateRateCard(rate); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, s.rewrite(`
INSERT INTO metering_rates (resource_type, price_per_unit, currency, unit) VALUES (?, ?, ?, ?)
ON CONFLICT(resource_type) DO UPDATE SET price_per_unit = excluded.price_per_unit, currency = excluded.currency, unit = excluded.unit
`), rate.ResourceType, rate.PricePerUnit, rate.Currency, rate.Unit)
	if err != nil {
		return pkgerrors.Internal("failed to set rate", err)
	}
	return nil
}

// ListRates returns all rate cards.
func (s *Store) ListRates(ctx context.Context) ([]metering.RateCard, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT resource_type, price_per_unit, currency, unit FROM metering_rates`)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list rates", err)
	}
	defer rows.Close()
	out := make([]metering.RateCard, 0)
	for rows.Next() {
		var rate metering.RateCard
		if err := rows.Scan(&rate.ResourceType, &rate.PricePerUnit, &rate.Currency, &rate.Unit); err != nil {
			return nil, pkgerrors.Internal("failed to scan rate", err)
		}
		out = append(out, rate)
	}
	return out, rows.Err()
}

// CalculateCost estimates cost for usage.
func (s *Store) CalculateCost(ctx context.Context, usage metering.UsageEvent) (float64, error) {
	if err := s.checkClosed(); err != nil {
		return 0, err
	}
	if err := metering.ValidateUsageEvent(usage); err != nil {
		return 0, err
	}
	rate, err := s.GetRate(ctx, usage.ResourceType)
	if err != nil {
		return 0, err
	}
	return usage.Quantity * rate.PricePerUnit, nil
}

// Close marks the store closed.
func (s *Store) Close() error {
	s.closed.Store(true)
	return nil
}
