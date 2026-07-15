/*
Package cassandra implements kv.KV against Apache Cassandra via gocql.

Production callers use New (real gocql session). Unit tests inject a
SessionAPI mock via NewFromSession. Integration against a live cluster
is skipped under -short (set CASSANDRA_HOST to enable).
*/
package cassandra
