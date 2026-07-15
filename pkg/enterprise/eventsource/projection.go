package eventsource

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/cqrs"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
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

// ProjectionMetrics receives optional instrumentation hooks from ProjectionRunner.
type ProjectionMetrics interface {
	// OnBatch records events applied in a RunOnce batch (matched + advanced).
	OnBatch(name string, applied int, advanced int64)
	// OnError records a projection failure (before backoff/retry).
	OnError(name string, err error)
	// OnCatchUpIdle is called when RunOnce finds nothing new to project.
	OnCatchUpIdle(name string)
}

// ProjectionConfig configures a ProjectionRunner.
type ProjectionConfig struct {
	// Name is the checkpoint key (defaults to first EventTypes entry or "default").
	Name string

	// BatchCheckpoints saves the checkpoint every N events (default 1).
	BatchCheckpoints int

	// PollInterval is the delay between successful RunOnce calls in Run (default 1s).
	PollInterval time.Duration

	// InitialBackoff is the first delay after a RunOnce error (default 200ms).
	InitialBackoff time.Duration

	// MaxBackoff caps exponential backoff after errors (default 30s).
	MaxBackoff time.Duration

	// Metrics optional hooks (nil-safe).
	Metrics ProjectionMetrics
}

// DefaultProjectionConfig returns sensible continuous-run defaults.
func DefaultProjectionConfig() ProjectionConfig {
	return ProjectionConfig{
		BatchCheckpoints: 1,
		PollInterval:     time.Second,
		InitialBackoff:   200 * time.Millisecond,
		MaxBackoff:       30 * time.Second,
	}
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
	def := DefaultProjectionConfig()
	if cfg.BatchCheckpoints <= 0 {
		cfg.BatchCheckpoints = def.BatchCheckpoints
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = def.PollInterval
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = def.InitialBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = def.MaxBackoff
	}
	if cfg.Name == "" {
		if types := projector.EventTypes(); len(types) > 0 {
			cfg.Name = types[0]
		} else {
			cfg.Name = "default"
		}
	}
	typeSet := make(map[string]struct{})
	if projector != nil {
		for _, t := range projector.EventTypes() {
			typeSet[t] = struct{}{}
		}
	}
	return &ProjectionRunner{
		store:       store,
		checkpoints: checkpoints,
		projector:   projector,
		cfg:         cfg,
		types:       typeSet,
	}
}

// Name returns the checkpoint / projection name.
func (r *ProjectionRunner) Name() string { return r.cfg.Name }

// Checkpoint returns the current durable checkpoint.
func (r *ProjectionRunner) Checkpoint(ctx context.Context) (Checkpoint, error) {
	if r.checkpoints == nil {
		return Checkpoint{}, ErrInvalidArgument("checkpoint store is required", nil)
	}
	return r.checkpoints.Load(ctx, r.cfg.Name)
}

// ResetCheckpoint forces the runner to restart from position 0 on the next RunOnce.
func (r *ProjectionRunner) ResetCheckpoint(ctx context.Context) error {
	if r.checkpoints == nil {
		return ErrInvalidArgument("checkpoint store is required", nil)
	}
	return r.checkpoints.Save(ctx, Checkpoint{
		Name:      r.cfg.Name,
		Position:  0,
		UpdatedAt: time.Now().UTC(),
	})
}

// RunOnce loads all events after the checkpoint, applies matching ones to the
// projector, and advances the checkpoint. It is safe to call repeatedly for
// catch-up; callers that need continuous projection should use Run.
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

	if start >= len(all) {
		if r.cfg.Metrics != nil {
			r.cfg.Metrics.OnCatchUpIdle(r.cfg.Name)
		}
		return nil
	}

	applied := 0
	sinceLastSave := 0
	for i := start; i < len(all); i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		ev := all[i]
		if r.shouldProject(ev.EventType) {
			if err := r.projector.Project(ctx, ev); err != nil {
				if r.cfg.Metrics != nil {
					r.cfg.Metrics.OnError(r.cfg.Name, err)
				}
				return ErrApplyFailed("projection failed for "+ev.EventType, err)
			}
			applied++
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
	if r.cfg.Metrics != nil {
		r.cfg.Metrics.OnBatch(r.cfg.Name, applied, cp.Position)
	}
	return nil
}

// Run continuously catch-up projects until ctx is cancelled.
// On success it waits PollInterval; on error it exponential-backoffs up to MaxBackoff
// then retries from the durable checkpoint (restart-safe).
func (r *ProjectionRunner) Run(ctx context.Context) error {
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := r.RunOnce(ctx)
		if err == nil {
			attempt = 0
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(r.cfg.PollInterval):
			}
			continue
		}
		if r.cfg.Metrics != nil {
			r.cfg.Metrics.OnError(r.cfg.Name, err)
		}
		delay := resilience.ExponentialBackoff(attempt, r.cfg.InitialBackoff, r.cfg.MaxBackoff, 0.1)
		attempt++
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
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
