package eventsource

import (
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/cqrs"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config configures eventsource stores and projection runners via env tags.
type Config struct {
	// StoreDriver selects the event store: "memory" (default). SQL/Postgres use adapter New.
	StoreDriver string `env:"EVENTSOURCE_STORE" env-default:"memory"`

	// CheckpointDriver selects checkpoint backend: "memory" (default).
	CheckpointDriver string `env:"EVENTSOURCE_CHECKPOINT" env-default:"memory"`

	// ProjectionName is the default checkpoint key for NewProjectionFromConfig.
	ProjectionName string `env:"EVENTSOURCE_PROJECTION_NAME" env-default:"default"`

	// BatchCheckpoints saves every N events (default 1).
	BatchCheckpoints int `env:"EVENTSOURCE_BATCH_CHECKPOINTS" env-default:"1"`

	// PollInterval between successful RunOnce cycles.
	PollInterval time.Duration `env:"EVENTSOURCE_POLL_INTERVAL" env-default:"1s"`

	// InitialBackoff after projection errors.
	InitialBackoff time.Duration `env:"EVENTSOURCE_INITIAL_BACKOFF" env-default:"200ms"`

	// MaxBackoff caps error backoff.
	MaxBackoff time.Duration `env:"EVENTSOURCE_MAX_BACKOFF" env-default:"30s"`
}

// ProjectionParts are the collaborators needed to build a ProjectionRunner.
type ProjectionParts struct {
	Store       EventStore
	Checkpoints CheckpointStore
	Projector   cqrs.Projector
}

// NewProjectionFromConfig builds a ProjectionRunner from Config + parts.
// StoreDriver/CheckpointDriver must already be wired into Parts (adapters are
// caller-owned); Config supplies projection timing and name defaults.
func NewProjectionFromConfig(cfg Config, parts ProjectionParts) (*ProjectionRunner, error) {
	if parts.Store == nil {
		return nil, pkgerrors.InvalidArgument("event store is required", nil)
	}
	if parts.Checkpoints == nil {
		return nil, pkgerrors.InvalidArgument("checkpoint store is required", nil)
	}
	if parts.Projector == nil {
		return nil, pkgerrors.InvalidArgument("projector is required", nil)
	}
	pcfg := ProjectionConfig{
		Name:             cfg.ProjectionName,
		BatchCheckpoints: cfg.BatchCheckpoints,
		PollInterval:     cfg.PollInterval,
		InitialBackoff:   cfg.InitialBackoff,
		MaxBackoff:       cfg.MaxBackoff,
	}
	return NewProjectionRunner(parts.Store, parts.Checkpoints, parts.Projector, pcfg), nil
}
