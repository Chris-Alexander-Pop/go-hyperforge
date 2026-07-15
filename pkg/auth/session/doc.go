// Package session provides distributed session management.
//
// This package defines the session manager interface and supports multiple storage backends.
// The default implementation is in-memory, but it is designed to support Redis, Memcached, etc.
//
// Set Config.EncryptionKey so session metadata is encrypted at rest (AES-GCM).
// Without it, metadata is stored in plaintext. Password credentials should use
// pkg/auth/password (crypto.Hasher), not session metadata.
package session
