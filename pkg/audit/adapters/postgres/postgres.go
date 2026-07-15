/*
Package postgres is a thin convenience wrapper around adapters/sql configured
for PostgreSQL placeholder style ($1, $2, ...).

Callers supply an open *sql.DB (e.g. lib/pq or pgx stdlib). Use adapters/sql
directly for SQLite tests or other dialects.
*/
package postgres

import (
	"database/sql"

	auditsql "github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/sql"
)

// Config mirrors auditsql.Config without Dialect (always Postgres).
type Config struct {
	HashChain bool
}

// New returns an auditsql.Store configured for PostgreSQL.
func New(db *sql.DB, cfg Config) (*auditsql.Store, error) {
	return auditsql.New(db, auditsql.Config{
		Dialect:   auditsql.DialectPostgres,
		HashChain: cfg.HashChain,
	})
}
