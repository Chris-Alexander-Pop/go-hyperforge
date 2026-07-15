package scheduler

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
)

// Strategy names for Config.Strategy.
const (
	StrategyRandom  = "random"
	StrategyBinpack = "binpack"
	StrategySpread  = "spread"
)

// Scheduler defines the interface for selecting optimal hosts for workloads.
type Scheduler interface {
	// SelectHost chooses the best host for the given requirements.
	// Returns the ID of the selected host.
	SelectHost(ctx context.Context, req Requirement) (string, error)
}

// Requirement defines the resource needs for a workload.
type Requirement struct {
	Resources    cloud.Resources   `json:"resources"`
	Region       string            `json:"region"`
	Zone         string            `json:"zone"`
	Tags         map[string]string `json:"tags,omitempty"`
	AntiAffinity []string          `json:"anti_affinity,omitempty"` // List of tag keys to avoid colocation
}

// Config holds configuration for the Scheduler.
type Config struct {
	// Strategy specifies the scheduling algorithm: "spread", "binpack", "random".
	Strategy string `env:"SCHEDULER_STRATEGY" env-default:"random"`
}
