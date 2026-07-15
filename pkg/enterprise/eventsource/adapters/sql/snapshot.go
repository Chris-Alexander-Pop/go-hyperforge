package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
)

var _ eventsource.SnapshotStore = (*SnapshotStore)(nil)

// SnapshotStore persists aggregate snapshots via database/sql.
type SnapshotStore struct {
	db      *sql.DB
	dialect Dialect
	table   string
}

// NewSnapshotStore wraps an existing *sql.DB. Call Migrate before use.
// Default table is aggregate_snapshots.
func NewSnapshotStore(db *sql.DB, cfg Config) (*SnapshotStore, error) {
	if db == nil {
		return nil, eventsource.ErrInvalidArgument("db is required", nil)
	}
	table := cfg.Table
	if table == "" {
		table = "aggregate_snapshots"
	}
	return &SnapshotStore{db: db, dialect: cfg.Dialect, table: table}, nil
}

func (s *SnapshotStore) rewrite(query string) string {
	return rewrite(s.dialect, query)
}

// Migrate creates the snapshot table if missing.
func (s *SnapshotStore) Migrate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	aggregate_id TEXT PRIMARY KEY,
	aggregate_type TEXT NOT NULL,
	version BIGINT NOT NULL,
	timestamp TIMESTAMP NOT NULL,
	data TEXT NOT NULL
)`, s.table)
	_, err := s.db.ExecContext(ctx, stmt)
	if err != nil {
		return eventsource.ErrApplyFailed("migrate aggregate snapshots", err)
	}
	return nil
}

// Save upserts a snapshot by aggregate_id (keeps the latest only).
func (s *SnapshotStore) Save(ctx context.Context, snapshot eventsource.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if snapshot.AggregateID == "" {
		return eventsource.ErrInvalidArgument("aggregateID is required", nil)
	}
	if snapshot.Timestamp.IsZero() {
		snapshot.Timestamp = time.Now().UTC()
	}
	data := snapshot.Data
	if data == nil {
		data = json.RawMessage("null")
	}

	var q string
	if s.dialect == DialectPostgres {
		q = fmt.Sprintf(`INSERT INTO %s (aggregate_id, aggregate_type, version, timestamp, data)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (aggregate_id) DO UPDATE SET
	aggregate_type = EXCLUDED.aggregate_type,
	version = EXCLUDED.version,
	timestamp = EXCLUDED.timestamp,
	data = EXCLUDED.data`, s.table)
	} else {
		q = fmt.Sprintf(`INSERT INTO %s (aggregate_id, aggregate_type, version, timestamp, data)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(aggregate_id) DO UPDATE SET
	aggregate_type = excluded.aggregate_type,
	version = excluded.version,
	timestamp = excluded.timestamp,
	data = excluded.data`, s.table)
	}
	_, err := s.db.ExecContext(ctx, q,
		snapshot.AggregateID,
		snapshot.AggregateType,
		snapshot.Version,
		snapshot.Timestamp.UTC(),
		[]byte(data),
	)
	if err != nil {
		return eventsource.ErrApplyFailed("save snapshot", err)
	}
	return nil
}

// Load retrieves the latest snapshot for an aggregate.
// Returns nil, nil when no snapshot exists (matches memory adapter semantics).
func (s *SnapshotStore) Load(ctx context.Context, aggregateID string) (*eventsource.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q := s.rewrite(fmt.Sprintf(
		`SELECT aggregate_id, aggregate_type, version, timestamp, data FROM %s WHERE aggregate_id = ?`,
		s.table,
	))
	var snap eventsource.Snapshot
	var data []byte
	err := s.db.QueryRowContext(ctx, q, aggregateID).Scan(
		&snap.AggregateID,
		&snap.AggregateType,
		&snap.Version,
		&snap.Timestamp,
		&data,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, eventsource.ErrApplyFailed("load snapshot", err)
	}
	snap.Data = json.RawMessage(data)
	return &snap, nil
}
