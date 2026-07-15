/*
Package memory provides an in-memory audit.Store for tests and local development.

Uses pkg/concurrency.SmartRWMutex for observability-friendly locking.
Events are retained only for the process lifetime.

Optional hash chaining (NewChainedStore / WithHashChain) stamps ID, Hash, and
PrevHash for tamper-evident append-only logs. Implements LifecycleStore for
retention purge and GDPR Export/Erase by actor ID.
*/
package memory
