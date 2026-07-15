/*
Package sql provides a durable audit.Store backed by database/sql.

Works with PostgreSQL and SQLite (and other dialects that accept the shared
DDL). Call Migrate once at startup. Optional hash chaining stamps prev_hash /
hash columns for tamper evidence. Implements LifecycleStore for retention
purge and GDPR Export/Erase by actor ID.
*/
package sql
