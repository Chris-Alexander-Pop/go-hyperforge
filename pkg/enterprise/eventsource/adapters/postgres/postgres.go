/*
Package postgres is a thin convenience wrapper around adapters/sql configured
for PostgreSQL placeholder style ($1, $2, ...).

Callers supply an open *sql.DB (e.g. lib/pq or pgx stdlib).
*/
package postgres

import (
	"database/sql"

	eventsql "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource/adapters/sql"
)

// Config mirrors eventsql.Config without Dialect (always Postgres).
type Config struct {
	Table string
}

// New returns an eventsql.CheckpointStore configured for PostgreSQL.
func New(db *sql.DB, cfg Config) (*eventsql.CheckpointStore, error) {
	return eventsql.New(db, eventsql.Config{
		Dialect: eventsql.DialectPostgres,
		Table:   cfg.Table,
	})
}
