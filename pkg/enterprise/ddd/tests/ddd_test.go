package ddd_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/enterprise/ddd"
)

type person struct {
	age int
}

type minAgeSpec struct{ min int }

func (s minAgeSpec) IsSatisfiedBy(p person) bool { return p.age >= s.min }

type maxAgeSpec struct{ max int }

func (s maxAgeSpec) IsSatisfiedBy(p person) bool { return p.age <= s.max }

func TestSpecifications(t *testing.T) {
	adult := minAgeSpec{min: 18}
	seniorCap := maxAgeSpec{max: 65}
	p := person{age: 30}

	if !adult.IsSatisfiedBy(p) {
		t.Fatal("expected adult")
	}
	if !ddd.And[person](adult, seniorCap).IsSatisfiedBy(p) {
		t.Fatal("expected And satisfied")
	}
	if ddd.And[person](adult, maxAgeSpec{max: 20}).IsSatisfiedBy(p) {
		t.Fatal("expected And not satisfied")
	}
	if !ddd.Or[person](adult, maxAgeSpec{max: 10}).IsSatisfiedBy(p) {
		t.Fatal("expected Or satisfied")
	}
	if ddd.Not[person](adult).IsSatisfiedBy(p) {
		t.Fatal("expected Not not satisfied")
	}
	if !ddd.Not[person](minAgeSpec{min: 40}).IsSatisfiedBy(p) {
		t.Fatal("expected Not satisfied")
	}
}

func TestEntityAndAggregate(t *testing.T) {
	e := ddd.NewBaseEntity()
	if e.ID() == "" {
		t.Fatal("expected generated id")
	}
	if e.CreatedAt().IsZero() || e.UpdatedAt().IsZero() {
		t.Fatal("expected timestamps")
	}
	e.Touch()

	named := ddd.NewBaseEntityWithID("fixed")
	if named.ID() != "fixed" {
		t.Fatalf("id want fixed got %s", named.ID())
	}

	agg := ddd.NewAggregateRoot()
	if agg.Version() != 0 {
		t.Fatalf("version want 0 got %d", agg.Version())
	}
	evt := ddd.NewDomainEvent("OrderPlaced", agg.ID(), "Order", 1)
	agg.AddDomainEvent(evt)
	if len(agg.GetUncommittedEvents()) != 1 {
		t.Fatal("expected 1 uncommitted event")
	}
	if evt.EventType() != "OrderPlaced" || evt.AggregateType() != "Order" || evt.Version() != 1 {
		t.Fatalf("unexpected event: %+v", evt)
	}
	agg.IncrementVersion()
	if agg.Version() != 1 {
		t.Fatal("expected version 1")
	}
	agg.ClearUncommittedEvents()
	if len(agg.GetUncommittedEvents()) != 0 {
		t.Fatal("expected cleared events")
	}
}

// memoryRepo verifies Repository[T] requires context on all methods.
type memoryRepo struct {
	items map[string]*ddd.BaseEntity
}

func (r *memoryRepo) FindByID(ctx context.Context, id string) (*ddd.BaseEntity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.items[id], nil
}

func (r *memoryRepo) Save(ctx context.Context, entity *ddd.BaseEntity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.items[entity.ID()] = entity
	return nil
}

func (r *memoryRepo) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(r.items, id)
	return nil
}

func TestRepositoryContext(t *testing.T) {
	var _ ddd.Repository[*ddd.BaseEntity] = (*memoryRepo)(nil)

	repo := &memoryRepo{items: map[string]*ddd.BaseEntity{}}
	ctx := context.Background()
	e := ddd.NewBaseEntityWithID("e1")
	if err := repo.Save(ctx, &e); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.FindByID(ctx, "e1")
	if err != nil || got == nil || got.ID() != "e1" {
		t.Fatalf("FindByID: %v %v", got, err)
	}
	if err := repo.Delete(ctx, "e1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
