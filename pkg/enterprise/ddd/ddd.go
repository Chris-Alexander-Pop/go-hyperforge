package ddd

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Entity is the base interface for domain entities with identity.
type Entity interface {
	ID() string
}

// BaseEntity provides common entity functionality.
type BaseEntity struct {
	id        string
	createdAt time.Time
	updatedAt time.Time
}

// NewBaseEntity creates a new base entity with a generated ID.
func NewBaseEntity() BaseEntity {
	return BaseEntity{
		id:        uuid.NewString(),
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// NewBaseEntityWithID creates a base entity with a specific ID.
func NewBaseEntityWithID(id string) BaseEntity {
	return BaseEntity{
		id:        id,
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// ID returns the entity identifier.
func (e *BaseEntity) ID() string {
	return e.id
}

// CreatedAt returns when the entity was created.
func (e *BaseEntity) CreatedAt() time.Time {
	return e.createdAt
}

// UpdatedAt returns when the entity was last updated.
func (e *BaseEntity) UpdatedAt() time.Time {
	return e.updatedAt
}

// Touch updates the updatedAt timestamp.
func (e *BaseEntity) Touch() {
	e.updatedAt = time.Now()
}

// ValueObject is the base interface for value objects (immutable, no identity).
type ValueObject interface {
	Equals(other ValueObject) bool
}

// DomainEvent represents something that happened in the domain.
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
	AggregateType() string
	Version() int
}

// BaseDomainEvent provides common event functionality.
type BaseDomainEvent struct {
	eventType     string
	occurredAt    time.Time
	aggregateID   string
	aggregateType string
	version       int
}

// NewDomainEvent creates a new domain event.
func NewDomainEvent(eventType, aggregateID, aggregateType string, version int) BaseDomainEvent {
	return BaseDomainEvent{
		eventType:     eventType,
		occurredAt:    time.Now(),
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		version:       version,
	}
}

func (e BaseDomainEvent) EventType() string     { return e.eventType }
func (e BaseDomainEvent) OccurredAt() time.Time { return e.occurredAt }
func (e BaseDomainEvent) AggregateID() string   { return e.aggregateID }
func (e BaseDomainEvent) AggregateType() string { return e.aggregateType }
func (e BaseDomainEvent) Version() int          { return e.version }

// AggregateRoot is the base for aggregate roots that manage domain events.
type AggregateRoot struct {
	BaseEntity
	version           int
	uncommittedEvents []DomainEvent
}

// NewAggregateRoot creates a new aggregate root.
func NewAggregateRoot() AggregateRoot {
	return AggregateRoot{
		BaseEntity:        NewBaseEntity(),
		version:           0,
		uncommittedEvents: make([]DomainEvent, 0),
	}
}

// Version returns the aggregate version.
func (a *AggregateRoot) Version() int {
	return a.version
}

// IncrementVersion increments the version.
func (a *AggregateRoot) IncrementVersion() {
	a.version++
}

// AddDomainEvent adds an event to the uncommitted events.
func (a *AggregateRoot) AddDomainEvent(event DomainEvent) {
	a.uncommittedEvents = append(a.uncommittedEvents, event)
}

// GetUncommittedEvents returns uncommitted domain events.
func (a *AggregateRoot) GetUncommittedEvents() []DomainEvent {
	return a.uncommittedEvents
}

// ClearUncommittedEvents clears the uncommitted events after persistence.
func (a *AggregateRoot) ClearUncommittedEvents() {
	a.uncommittedEvents = make([]DomainEvent, 0)
}

// Repository is the generic repository pattern interface.
// All methods take context.Context as the first parameter for cancellation,
// deadlines, and tracing propagation (PACKAGE_STANDARDS §1.4).
type Repository[T Entity] interface {
	FindByID(ctx context.Context, id string) (T, error)
	Save(ctx context.Context, entity T) error
	Delete(ctx context.Context, id string) error
}

// Specification is the specification pattern for querying.
type Specification[T any] interface {
	IsSatisfiedBy(entity T) bool
}

// AndSpecification combines two specifications with AND.
type AndSpecification[T any] struct {
	left  Specification[T]
	right Specification[T]
}

// And creates an AND specification.
func And[T any](left, right Specification[T]) *AndSpecification[T] {
	return &AndSpecification[T]{left: left, right: right}
}

// IsSatisfiedBy checks if both specifications are satisfied.
func (s *AndSpecification[T]) IsSatisfiedBy(entity T) bool {
	return s.left.IsSatisfiedBy(entity) && s.right.IsSatisfiedBy(entity)
}

// OrSpecification combines two specifications with OR.
type OrSpecification[T any] struct {
	left  Specification[T]
	right Specification[T]
}

// Or creates an OR specification.
func Or[T any](left, right Specification[T]) *OrSpecification[T] {
	return &OrSpecification[T]{left: left, right: right}
}

// IsSatisfiedBy checks if either specification is satisfied.
func (s *OrSpecification[T]) IsSatisfiedBy(entity T) bool {
	return s.left.IsSatisfiedBy(entity) || s.right.IsSatisfiedBy(entity)
}

// NotSpecification negates a specification.
type NotSpecification[T any] struct {
	spec Specification[T]
}

// Not creates a NOT specification.
func Not[T any](spec Specification[T]) *NotSpecification[T] {
	return &NotSpecification[T]{spec: spec}
}

// IsSatisfiedBy checks if the specification is not satisfied.
func (s *NotSpecification[T]) IsSatisfiedBy(entity T) bool {
	return !s.spec.IsSatisfiedBy(entity)
}

// DomainService is a marker interface for domain services.
type DomainService interface{}

// Factory is the factory pattern interface.
type Factory[T any] interface {
	Create() T
}
