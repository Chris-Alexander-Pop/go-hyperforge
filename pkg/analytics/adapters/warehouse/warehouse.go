package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/bigdata"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Ensure Sink implements analytics.Sink.
var _ analytics.Sink = (*Sink)(nil)

// Config configures the warehouse sink.
type Config struct {
	// Table is the fully-qualified table name used in INSERT statements
	// (e.g. "analytics.events" or "events"). Required.
	Table string

	// Client is the bigdata warehouse client. Required.
	Client bigdata.Client
}

// Sink ingests analytics events by INSERT-ing rows via bigdata.Client.Query.
type Sink struct {
	client bigdata.Client
	table  string
	closed atomic.Bool
}

// New creates a warehouse event sink.
func New(cfg Config) (*Sink, error) {
	if cfg.Client == nil {
		return nil, errors.InvalidArgument("warehouse Client is required", nil)
	}
	if cfg.Table == "" {
		return nil, errors.InvalidArgument("warehouse Table is required", nil)
	}
	return &Sink{client: cfg.Client, table: cfg.Table}, nil
}

// Ingest writes events as INSERT statements through the bigdata client.
func (s *Sink) Ingest(ctx context.Context, events ...analytics.Event) error {
	if s == nil || s.closed.Load() {
		return analytics.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	for _, e := range events {
		if e.Timestamp.IsZero() {
			e.Timestamp = time.Now().UTC()
		}
		propsJSON := "{}"
		if e.Properties != nil {
			b, err := json.Marshal(e.Properties)
			if err != nil {
				return errors.InvalidArgument("failed to marshal event properties", err)
			}
			propsJSON = string(b)
		}

		// Positional args keep adapters (Snowflake SQL, etc.) safe from injection.
		query := fmt.Sprintf(
			`INSERT INTO %s (name, user_id, properties, timestamp) VALUES (?, ?, ?, ?)`,
			s.table,
		)
		if _, err := s.client.Query(ctx, query, e.Name, e.UserID, propsJSON, e.Timestamp.UTC()); err != nil {
			return errors.Internal("warehouse ingest failed", err)
		}
	}
	return nil
}

// Close marks the sink closed. It does not close the underlying bigdata.Client
// (caller owns the client lifecycle).
func (s *Sink) Close() error {
	if s == nil {
		return nil
	}
	s.closed.Store(true)
	return nil
}
