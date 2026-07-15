package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/audit"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
	"github.com/google/uuid"
)

// Ensure compile-time interface compliance.
var (
	_ audit.Store          = (*Store)(nil)
	_ audit.LifecycleStore = (*Store)(nil)
)

// Dialect selects SQL placeholder style.
type Dialect int

const (
	// DialectSQLite uses ? placeholders (also used by MySQL-style drivers).
	DialectSQLite Dialect = iota
	// DialectPostgres uses $1, $2, ... placeholders.
	DialectPostgres
)

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS audit_events (
	id TEXT PRIMARY KEY,
	timestamp TIMESTAMP NOT NULL,
	event_type TEXT NOT NULL,
	outcome TEXT NOT NULL DEFAULT '',
	actor_id TEXT NOT NULL DEFAULT '',
	actor_type TEXT NOT NULL DEFAULT '',
	actor_ip TEXT NOT NULL DEFAULT '',
	actor_user_agent TEXT NOT NULL DEFAULT '',
	target_id TEXT NOT NULL DEFAULT '',
	target_type TEXT NOT NULL DEFAULT '',
	resource_id TEXT NOT NULL DEFAULT '',
	resource_type TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	metadata TEXT NOT NULL DEFAULT '{}',
	request_id TEXT NOT NULL DEFAULT '',
	session_id TEXT NOT NULL DEFAULT '',
	correlation_id TEXT NOT NULL DEFAULT '',
	error_code TEXT NOT NULL DEFAULT '',
	error_message TEXT NOT NULL DEFAULT '',
	hash TEXT NOT NULL DEFAULT '',
	prev_hash TEXT NOT NULL DEFAULT ''
)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_events_actor ON audit_events(actor_id)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_events_type ON audit_events(event_type)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_events_ts ON audit_events(timestamp)`,
}

// Config configures the SQL audit store.
type Config struct {
	// Dialect selects placeholder style (SQLite ? vs Postgres $n).
	Dialect Dialect

	// HashChain enables tamper-evident chaining on Append.
	HashChain bool

	// Retrier wraps DB I/O; nil uses resilience.DefaultRetryConfig.
	Retrier resilience.Retrier
}

// Store is a durable audit store using database/sql.
type Store struct {
	db        *sql.DB
	dialect   Dialect
	hashChain bool
	retrier   resilience.Retrier
}

// New wraps an existing *sql.DB. Call Migrate before use.
func New(db *sql.DB, cfg Config) (*Store, error) {
	if db == nil {
		return nil, audit.ErrInvalidArgument("db is required", nil)
	}
	retrier := cfg.Retrier
	if retrier == nil {
		retrier = resilience.NewRetrier(resilience.DefaultRetryConfig())
	}
	return &Store{
		db:        db,
		dialect:   cfg.Dialect,
		hashChain: cfg.HashChain,
		retrier:   retrier,
	}, nil
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

// Migrate creates the audit_events table and indexes if missing.
func (s *Store) Migrate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.retrier.Execute(ctx, func(ctx context.Context) error {
		for _, stmt := range schemaStatements {
			if _, err := s.db.ExecContext(ctx, stmt); err != nil {
				return audit.ErrAppendFailed("migrate audit_events failed", err)
			}
		}
		return nil
	})
}

// Append inserts an audit event.
func (s *Store) Append(ctx context.Context, event audit.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.EventType == "" {
		return audit.ErrInvalidArgument("event_type is required", nil)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}

	return s.retrier.Execute(ctx, func(ctx context.Context) error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return audit.ErrAppendFailed("begin tx failed", err)
		}
		defer func() { _ = tx.Rollback() }()

		if s.hashChain {
			prev := "GENESIS"
			var last string
			q := s.rewrite(`SELECT hash FROM audit_events WHERE hash != '' ORDER BY timestamp DESC, id DESC LIMIT 1`)
			err := tx.QueryRowContext(ctx, q).Scan(&last)
			if err == nil && last != "" {
				prev = last
			} else if err != nil && err != sql.ErrNoRows {
				return audit.ErrAppendFailed("read chain tip failed", err)
			}
			event.PrevHash = prev
			h, err := audit.HashEvent(event)
			if err != nil {
				return err
			}
			event.Hash = h
		}

		meta, err := json.Marshal(event.Metadata)
		if err != nil {
			return audit.ErrMarshalFailed(err)
		}
		if meta == nil {
			meta = []byte("{}")
		}

		insert := s.rewrite(`
INSERT INTO audit_events (
	id, timestamp, event_type, outcome, actor_id, actor_type, actor_ip, actor_user_agent,
	target_id, target_type, resource_id, resource_type, action, description, metadata,
	request_id, session_id, correlation_id, error_code, error_message, hash, prev_hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		_, err = tx.ExecContext(ctx, insert,
			event.ID, event.Timestamp.UTC(), string(event.EventType), string(event.Outcome),
			event.ActorID, event.ActorType, event.ActorIP, event.ActorUserAgent,
			event.TargetID, event.TargetType, event.ResourceID, event.ResourceType,
			event.Action, event.Description, string(meta),
			event.RequestID, event.SessionID, event.CorrelationID, event.ErrorCode, event.ErrorMessage,
			event.Hash, event.PrevHash,
		)
		if err != nil {
			return audit.ErrAppendFailed("insert audit event failed", err)
		}
		return tx.Commit()
	})
}

// Query returns matching events ordered by timestamp ascending.
func (s *Store) Query(ctx context.Context, filter audit.QueryFilter) ([]audit.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if filter.Limit < 0 {
		return nil, audit.ErrInvalidArgument("limit must be >= 0", nil)
	}

	var out []audit.Event
	err := s.retrier.Execute(ctx, func(ctx context.Context) error {
		q := `SELECT id, timestamp, event_type, outcome, actor_id, actor_type, actor_ip, actor_user_agent,
			target_id, target_type, resource_id, resource_type, action, description, metadata,
			request_id, session_id, correlation_id, error_code, error_message, hash, prev_hash
			FROM audit_events WHERE 1=1`
		args := make([]interface{}, 0, 4)
		if filter.ActorID != "" {
			q += ` AND actor_id = ?`
			args = append(args, filter.ActorID)
		}
		if filter.EventType != "" {
			q += ` AND event_type = ?`
			args = append(args, string(filter.EventType))
		}
		if !filter.Since.IsZero() {
			q += ` AND timestamp >= ?`
			args = append(args, filter.Since.UTC())
		}
		if !filter.Until.IsZero() {
			q += ` AND timestamp <= ?`
			args = append(args, filter.Until.UTC())
		}
		q += ` ORDER BY timestamp ASC, id ASC`
		if filter.Limit > 0 {
			q += ` LIMIT ?`
			args = append(args, filter.Limit)
		}

		rows, err := s.db.QueryContext(ctx, s.rewrite(q), args...)
		if err != nil {
			return audit.ErrQueryFailed("", err)
		}
		defer rows.Close()

		events := make([]audit.Event, 0)
		for rows.Next() {
			e, err := scanEvent(rows)
			if err != nil {
				return err
			}
			events = append(events, e)
		}
		if err := rows.Err(); err != nil {
			return audit.ErrQueryFailed("", err)
		}
		out = events
		return nil
	})
	return out, err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanEvent(row rowScanner) (audit.Event, error) {
	var (
		e         audit.Event
		eventType string
		outcome   string
		metaRaw   string
		ts        time.Time
	)
	err := row.Scan(
		&e.ID, &ts, &eventType, &outcome, &e.ActorID, &e.ActorType, &e.ActorIP, &e.ActorUserAgent,
		&e.TargetID, &e.TargetType, &e.ResourceID, &e.ResourceType, &e.Action, &e.Description, &metaRaw,
		&e.RequestID, &e.SessionID, &e.CorrelationID, &e.ErrorCode, &e.ErrorMessage, &e.Hash, &e.PrevHash,
	)
	if err != nil {
		return audit.Event{}, audit.ErrQueryFailed("scan failed", err)
	}
	e.Timestamp = ts.UTC()
	e.EventType = audit.EventType(eventType)
	e.Outcome = audit.Outcome(outcome)
	if metaRaw != "" && metaRaw != "{}" {
		_ = json.Unmarshal([]byte(metaRaw), &e.Metadata)
	}
	return e, nil
}

// Purge deletes events older than olderThan.
func (s *Store) Purge(ctx context.Context, olderThan time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if olderThan.IsZero() {
		return 0, audit.ErrInvalidArgument("olderThan is required", nil)
	}
	var n int64
	err := s.retrier.Execute(ctx, func(ctx context.Context) error {
		res, err := s.db.ExecContext(ctx, s.rewrite(`DELETE FROM audit_events WHERE timestamp < ?`), olderThan.UTC())
		if err != nil {
			return audit.ErrPurgeFailed("", err)
		}
		n, err = res.RowsAffected()
		if err != nil {
			return audit.ErrPurgeFailed("rows affected", err)
		}
		return nil
	})
	return n, err
}

// ExportByActor returns all events for actorID.
func (s *Store) ExportByActor(ctx context.Context, actorID string) ([]audit.Event, error) {
	if actorID == "" {
		return nil, audit.ErrInvalidArgument("actorID is required", nil)
	}
	return s.Query(ctx, audit.QueryFilter{ActorID: actorID})
}

// EraseByActor permanently deletes events for actorID.
func (s *Store) EraseByActor(ctx context.Context, actorID string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if actorID == "" {
		return 0, audit.ErrInvalidArgument("actorID is required", nil)
	}
	var n int64
	err := s.retrier.Execute(ctx, func(ctx context.Context) error {
		res, err := s.db.ExecContext(ctx, s.rewrite(`DELETE FROM audit_events WHERE actor_id = ?`), actorID)
		if err != nil {
			return audit.ErrEraseFailed("", err)
		}
		n, err = res.RowsAffected()
		if err != nil {
			return audit.ErrEraseFailed("rows affected", err)
		}
		return nil
	})
	return n, err
}
