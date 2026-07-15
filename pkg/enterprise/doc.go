/*
Package enterprise provides Domain-Driven Design (DDD), CQRS, and Event Sourcing primitives.

# Subpackages

  - ddd: entities, value objects, aggregates, domain events, specifications
  - cqrs: command/query buses and handlers
  - eventsource: append-only event stores, snapshots, event-sourced aggregates

# How this relates to sibling packages

  - pkg/events: in-process Publish/Subscribe bus for decoupling components.
    enterprise/eventsource is an append-only aggregate history store; use
    EventedStore (or an events.Bus option) to fan committed stream events onto
    pkg/events after Append.
  - pkg/messaging: durable, broker-backed messaging (Kafka, NATS, SQS, …).
    Prefer messaging when projections or integrations cross process boundaries;
    pkg/events remains for local fan-out.
  - pkg/audit: compliance/security audit trail (who did what). Domain and
    stream events are not audit logs; emit audit events separately when needed.
  - pkg/workflow: long-running process / saga orchestration. Workflows may
    consume domain events or issue commands via cqrs; they are not a substitute
    for an event store.

Import a subpackage directly:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/ddd"
	import "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/cqrs"
	import "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
*/
package enterprise
