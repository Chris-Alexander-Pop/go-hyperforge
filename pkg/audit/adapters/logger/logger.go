/*
Package logger provides a stdout audit.Store sink backed by pkg/logger (slog).

Append writes structured audit fields to the process logger. Query is not
supported and returns audit.ErrNotSupported — use adapters/memory or a durable
backend when query/export is required.
*/
package logger

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/audit"
	pkglogger "github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
)

// Ensure compile-time interface compliance.
var _ audit.Store = (*Sink)(nil)

// Sink is a write-only audit store that logs events to stdout via slog.
type Sink struct {
	log *slog.Logger
}

// NewSink creates a stdout audit sink using the process default logger.
func NewSink() *Sink {
	return &Sink{log: pkglogger.L()}
}

// NewSinkWithLogger creates a stdout audit sink with an explicit slog.Logger.
func NewSinkWithLogger(log *slog.Logger) *Sink {
	if log == nil {
		log = pkglogger.L()
	}
	return &Sink{log: log}
}

// Append marshals and logs the audit event.
func (s *Sink) Append(ctx context.Context, event audit.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return audit.ErrMarshalFailed(err)
	}

	s.log.InfoContext(ctx, "audit",
		"event", string(data),
		"event_type", string(event.EventType),
		"outcome", string(event.Outcome),
		"actor_id", event.ActorID,
		"target_id", event.TargetID,
	)
	return nil
}

// Query is not supported by the stdout sink.
func (s *Sink) Query(ctx context.Context, filter audit.QueryFilter) ([]audit.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, audit.ErrNotSupported
}
