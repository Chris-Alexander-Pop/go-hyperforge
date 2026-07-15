package eventsource

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/cqrs"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
)

// Checkpoint records how far a projector has processed the event store.
//
// Position is a 0-based index into EventStore.LoadAll order: the next event
// to process is at index Position. After processing N events from an empty
// store, Position == N.
type Checkpoint struct {
	// Name identifies the projector (unique per runner/store).
	Name string `json:"name"`

	// Position is the number of LoadAll events already applied.
	Position int64 `json:"position"`

	// UpdatedAt is when the checkpoint was last saved.
	UpdatedAt time.Time `json:"updated_at"`
}

// CheckpointStore persists projection catch-up positions.
type CheckpointStore interface {
	// Load returns the checkpoint for name, or a zero checkpoint (Position 0)
	// when none exists.
	Load(ctx context.Context, name string) (Checkpoint, error)

	// Save upserts the checkpoint.
	Save(ctx context.Context, cp Checkpoint) error
}

// ProjectionConfig configures a ProjectionRunner.
type ProjectionConfig struct {
	// Name is the checkpoint key (defaults to first EventTypes entry or "default").
	Name string

	// BatchCheckpoints saves the checkpoint every N events (default 1).
	BatchCheckpoints int
}

// ProjectionRunner catch-up projects events from an EventStore onto a
// cqrs.Projector, advancing a durable CheckpointStore.
type ProjectionRunner struct {
	store       EventStore
	checkpoints CheckpointStore
	projector   cqrs.Projector
	cfg         ProjectionConfig
	types       map[string]struct{}
}

// NewProjectionRunner wires store + checkpoints + projector.
func NewProjectionRunner(store EventStore, checkpoints CheckpointStore, projector cqrs.Projector, cfg ProjectionConfig) *ProjectionRunner {
	if cfg.BatchCheckpoints <= 0 {
		cfg.BatchCheckpoints = 1
	}
	if cfg.Name == "" {
		if types := projector.EventTypes(); len(types) > 0 {
			cfg.Name = types[0]
		} else {
			cfg.Name = "default"
		}
	}
	typeSet := make(map[string]struct{})
	for _, t := range projector.EventTypes() {
		typeSet[t] = struct{}{}
	}
	return &ProjectionRunner{
		store:       store,
		checkpoints: checkpoints,
		projector:   projector,
		cfg:         cfg,
		types:       typeSet,
	}
}

// RunOnce loads all events after the checkpoint, applies matching ones to the
// projector, and advances the checkpoint. It is safe to call repeatedly for
// catch-up; callers that need continuous projection should loop or subscribe
// via EventedStore / messaging outbox.
func (r *ProjectionRunner) RunOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.store == nil {
		return ErrInvalidArgument("event store is required", nil)
	}
	if r.checkpoints == nil {
		return ErrInvalidArgument("checkpoint store is required", nil)
	}
	if r.projector == nil {
		return ErrInvalidArgument("projector is required", nil)
	}

	cp, err := r.checkpoints.Load(ctx, r.cfg.Name)
	if err != nil {
		return err
	}

	all, err := r.store.LoadAll(ctx)
	if err != nil {
		return err
	}

	start := int(cp.Position)
	if start < 0 {
		start = 0
	}
	if start > len(all) {
		// Store was compacted/reset; restart from beginning.
		start = 0
		cp.Position = 0
	}

	sinceLastSave := 0
	for i := start; i < len(all); i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		ev := all[i]
		if r.shouldProject(ev.EventType) {
			if err := r.projector.Project(ctx, ev); err != nil {
				return ErrApplyFailed("projection failed for "+ev.EventType, err)
			}
		}
		cp.Position = int64(i + 1)
		cp.Name = r.cfg.Name
		cp.UpdatedAt = time.Now().UTC()
		sinceLastSave++

		if sinceLastSave >= r.cfg.BatchCheckpoints || i == len(all)-1 {
			if err := r.checkpoints.Save(ctx, cp); err != nil {
				return err
			}
			sinceLastSave = 0
		}
	}
	return nil
}

func (r *ProjectionRunner) shouldProject(eventType string) bool {
	if len(r.types) == 0 {
		return true
	}
	_, ok := r.types[eventType]
	return ok
}

// NewEventedStoreWithOutbox wraps next so Append fans out to a local events.Bus
// and a messaging outbox (durable broker delivery for projections/integrations).
//
// bus may be nil; only the messaging outbox is used in that case via a
// thin adapter. producer must be non-nil.
func NewEventedStoreWithOutbox(next EventStore, bus events.Bus, producer messaging.Producer) *EventedStore {
	outbox := events.NewOutbox(producer)
	if bus == nil {
		return NewEventedStore(next, &outboxOnlyBus{outbox: outbox})
	}
	return NewEventedStore(next, events.NewOutboxBus(bus, outbox))
}

// outboxOnlyBus satisfies events.Bus for messaging-only fan-out.
type outboxOnlyBus struct {
	outbox *events.Outbox
}

func (b *outboxOnlyBus) Publish(ctx context.Context, topic string, event events.Event) error {
	return b.outbox.Publish(ctx, topic, event)
}

func (b *outboxOnlyBus) Subscribe(ctx context.Context, topic string, handler events.Handler) (events.Subscription, error) {
	return "", ErrInvalidArgument("outbox-only bus does not support Subscribe", nil)
}

func (b *outboxOnlyBus) Unsubscribe(ctx context.Context, id events.Subscription) error {
	return nil
}

func (b *outboxOnlyBus) Close() error { return nil }
