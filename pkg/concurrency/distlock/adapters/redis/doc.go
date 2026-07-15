/*
Package redis provides a Redis-based distributed lock adapter.

Uses a single Redis instance with SET NX (acquire) and Lua scripts for
atomic release/extend. This is not the Redlock multi-master algorithm.
*/
package redis
