/*
Package postgres stores controlplane host/instance inventory in database/sql.

Call Migrate after New. Use DialectSQLite (modernc.org/sqlite) in tests and
DialectPostgres in production.
*/
package postgres
