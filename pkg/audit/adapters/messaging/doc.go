/*
Package messaging provides an audit.Store decorator that fans out Append
to a pkg/messaging.Producer after (or instead of) writing to an inner store.

Typical use: durable SQL/memory store as primary + Kafka/memory messaging for
SIEM / analytics consumers. Query/Purge/Export/Erase delegate to the inner
LifecycleStore when available.
*/
package messaging
