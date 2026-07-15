/*
Package sql provides durable CheckpointStore and SnapshotStore adapters backed by database/sql.

Supports SQLite (? placeholders) and PostgreSQL ($n). Callers supply an open
*sql.DB (e.g. modernc.org/sqlite for tests, pgx/stdlib or lib/pq for Postgres).
*/
package sql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
)

var _ eventsource.CheckpointStore = (*CheckpointStore)(nil)

// Dialect selects SQL placeholder style.
type Dialect int

const (
	// DialectSQLite uses ? placeholders.
	DialectSQLite Dialect = iota
	// DialectPostgres uses $1, $2, ... placeholders.
	DialectPostgres
)

// Config configures the SQL checkpoint store.
type Config struct {
	// Dialect selects placeholder style (SQLite ? vs Postgres $n).
	Dialect Dialect

	// Table is the checkpoint table name (default projection_checkpoints).
	Table string
}

// CheckpointStore persists projection checkpoints via database/sql.
type CheckpointStore struct {
	db      *sql.DB
	dialect Dialect
	table   string
}

// New wraps an existing *sql.DB. Call Migrate before use.
func New(db *sql.DB, cfg Config) (*CheckpointStore, error) {
	if db == nil {
		return nil, eventsource.ErrInvalidArgument("db is required", nil)
	}
	table := cfg.Table
	if table == "" {
		table = "projection_checkpoints"
	}
	return &CheckpointStore{db: db, dialect: cfg.Dialect, table: table}, nil
}

func (s *CheckpointStore) rewrite(query string) string {
	return rewrite(s.dialect, query)
}

// Migrate creates the checkpoint table if missing.
func (s *CheckpointStore) Migrate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	name TEXT PRIMARY KEY,
	position BIGINT NOT NULL DEFAULT 0,
	updated_at TIMESTAMP NOT NULL
)`, s.table)
	_, err := s.db.ExecContext(ctx, stmt)
	if err != nil {
		return eventsource.ErrApplyFailed("migrate projection checkpoints", err)
	}
	return nil
}

// Load returns the checkpoint for name, or Position 0 when missing.
func (s *CheckpointStore) Load(ctx context.Context, name string) (eventsource.Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return eventsource.Checkpoint{}, err
	}
	q := s.rewrite(fmt.Sprintf(`SELECT name, position, updated_at FROM %s WHERE name = ?`, s.table))
	var cp eventsource.Checkpoint
	err := s.db.QueryRowContext(ctx, q, name).Scan(&cp.Name, &cp.Position, &cp.UpdatedAt)
	if err == sql.ErrNoRows {
		return eventsource.Checkpoint{Name: name}, nil
	}
	if err != nil {
		return eventsource.Checkpoint{}, eventsource.ErrApplyFailed("load checkpoint", err)
	}
	return cp, nil
}

// Save upserts the checkpoint.
func (s *CheckpointStore) Save(ctx context.Context, cp eventsource.Checkpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cp.Name == "" {
		return eventsource.ErrInvalidArgument("checkpoint name is required", nil)
	}
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = time.Now().UTC()
	}

	var q string
	if s.dialect == DialectPostgres {
		q = fmt.Sprintf(`INSERT INTO %s (name, position, updated_at) VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET position = EXCLUDED.position, updated_at = EXCLUDED.updated_at`, s.table)
	} else {
		q = fmt.Sprintf(`INSERT INTO %s (name, position, updated_at) VALUES (?, ?, ?)
ON CONFLICT(name) DO UPDATE SET position = excluded.position, updated_at = excluded.updated_at`, s.table)
	}
	_, err := s.db.ExecContext(ctx, q, cp.Name, cp.Position, cp.UpdatedAt.UTC())
	if err != nil {
		return eventsource.ErrApplyFailed("save checkpoint", err)
	}
	return nil
}
