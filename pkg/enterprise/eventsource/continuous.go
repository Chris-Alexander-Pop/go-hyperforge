package eventsource

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/cqrs"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// ContinuousProjector continuously projects either by polling an EventStore
// (catch-up via ProjectionRunner) or by consuming a messaging outbox Consumer.
// Prefer outbox mode when Append fans out via NewEventedStoreWithOutbox.
type ContinuousProjector struct {
	runner    *ProjectionRunner
	consumer  messaging.Consumer
	projector cqrs.Projector
	cfg       ProjectionConfig
	types     map[string]struct{}
}

// ContinuousProjectorConfig selects EventStore and/or outbox consumption.
type ContinuousProjectorConfig struct {
	ProjectionConfig

	// Store + Checkpoints enable EventStore catch-up looping (ProjectionRunner.Run).
	Store       EventStore
	Checkpoints CheckpointStore
	Projector   cqrs.Projector

	// Consumer enables outbox-driven projection (messaging.Consumer of outbox envelopes).
	Consumer messaging.Consumer
}

// NewContinuousProjector wires EventStore polling and/or messaging outbox consumption.
func NewContinuousProjector(cfg ContinuousProjectorConfig) (*ContinuousProjector, error) {
	if cfg.Projector == nil {
		return nil, ErrInvalidArgument("projector is required", nil)
	}
	if cfg.Store == nil && cfg.Consumer == nil {
		return nil, ErrInvalidArgument("event store or messaging consumer is required", nil)
	}
	def := DefaultProjectionConfig()
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
		if types := cfg.Projector.EventTypes(); len(types) > 0 {
			cfg.Name = types[0]
		} else {
			cfg.Name = "default"
		}
	}
	typeSet := make(map[string]struct{})
	for _, t := range cfg.Projector.EventTypes() {
		typeSet[t] = struct{}{}
	}

	cp := &ContinuousProjector{
		consumer:  cfg.Consumer,
		projector: cfg.Projector,
		cfg:       cfg.ProjectionConfig,
		types:     typeSet,
	}
	if cfg.Store != nil {
		if cfg.Checkpoints == nil {
			return nil, ErrInvalidArgument("checkpoint store is required with event store", nil)
		}
		cp.runner = NewProjectionRunner(cfg.Store, cfg.Checkpoints, cfg.Projector, cfg.ProjectionConfig)
	}
	return cp, nil
}

// Name returns the projection name.
func (c *ContinuousProjector) Name() string { return c.cfg.Name }

// Run loops until ctx is cancelled.
// When a messaging Consumer is configured it is preferred (outbox-driven).
// Otherwise EventStore catch-up via ProjectionRunner.Run is used.
func (c *ContinuousProjector) Run(ctx context.Context) error {
	if c.consumer != nil {
		return c.runOutbox(ctx)
	}
	if c.runner != nil {
		return c.runner.Run(ctx)
	}
	return ErrInvalidArgument("no projection source configured", nil)
}

// RunOutbox consumes messaging outbox envelopes and projects matching events
// with exponential backoff on handler errors.
func (c *ContinuousProjector) RunOutbox(ctx context.Context) error {
	if c.consumer == nil {
		return ErrInvalidArgument("messaging consumer is required", nil)
	}
	return c.runOutbox(ctx)
}

func (c *ContinuousProjector) runOutbox(ctx context.Context) error {
	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := c.consumer.Consume(ctx, func(msgCtx context.Context, msg *messaging.Message) error {
			if err := c.handleOutboxMessage(msgCtx, msg); err != nil {
				if c.cfg.Metrics != nil {
					c.cfg.Metrics.OnError(c.cfg.Name, err)
				}
				return err
			}
			attempt = 0
			return nil
		})
		if err == nil || ctx.Err() != nil {
			return err
		}
		if c.cfg.Metrics != nil {
			c.cfg.Metrics.OnError(c.cfg.Name, err)
		}
		delay := resilience.ExponentialBackoff(attempt, c.cfg.InitialBackoff, c.cfg.MaxBackoff, 0.1)
		attempt++
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (c *ContinuousProjector) handleOutboxMessage(ctx context.Context, msg *messaging.Message) error {
	if msg == nil {
		return nil
	}
	var env events.OutboxPayload
	if err := json.Unmarshal(msg.Payload, &env); err != nil {
		var ev Event
		if err2 := json.Unmarshal(msg.Payload, &ev); err2 != nil {
			return ErrApplyFailed("invalid outbox payload", err)
		}
		return c.projectEvent(ctx, ev)
	}
	// Prefer nested eventsource.Event when outbox payload is the full stored event.
	var nested Event
	if len(env.Payload) > 0 && json.Unmarshal(env.Payload, &nested) == nil && nested.EventType != "" {
		return c.projectEvent(ctx, nested)
	}
	ev := Event{
		ID:        env.ID,
		EventType: env.Type,
		Timestamp: env.Timestamp,
		Data:      env.Payload,
	}
	if ev.EventType == "" {
		ev.EventType = msg.Headers["x-events-type"]
	}
	if ev.AggregateType == "" {
		ev.AggregateType = env.Topic
	}
	return c.projectEvent(ctx, ev)
}

func (c *ContinuousProjector) projectEvent(ctx context.Context, ev Event) error {
	if !c.shouldProject(ev.EventType) {
		return nil
	}
	if err := c.projector.Project(ctx, ev); err != nil {
		return ErrApplyFailed("projection failed for "+ev.EventType, err)
	}
	if c.cfg.Metrics != nil {
		c.cfg.Metrics.OnBatch(c.cfg.Name, 1, 0)
	}
	return nil
}

func (c *ContinuousProjector) shouldProject(eventType string) bool {
	if len(c.types) == 0 {
		return true
	}
	_, ok := c.types[eventType]
	return ok
}

// Runner returns the underlying ProjectionRunner when EventStore mode is enabled.
func (c *ContinuousProjector) Runner() *ProjectionRunner { return c.runner }
