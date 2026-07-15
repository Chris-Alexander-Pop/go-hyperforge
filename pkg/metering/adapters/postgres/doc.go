/*
Package postgres persists metering usage events and rate cards via database/sql.

Call Migrate after New. DialectSQLite for tests; DialectPostgres for production.
*/
package postgres
