package enterprise

// This umbrella package documents the enterprise pattern family.
// Concrete APIs live in the ddd, cqrs, and eventsource subpackages.
//
// Boundaries (see also doc.go):
//   - eventsource: durable-style aggregate event streams (persistence model)
//   - pkg/events: local in-process pub/sub
//   - pkg/messaging: distributed brokers
//   - pkg/audit: compliance audit records
//   - pkg/workflow: process/saga orchestration
