// Package distributed provides configuration helpers for distributed training.
//
// Supports TorchRun (Elastic) and DeepSpeed configuration generation.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training/distributed"
//
//	dsConfig := distributed.NewDeepSpeedConfig(distributed.ZeroStage2)
//	cmd := distributed.TorchRunCommand("train.py", 4)
package distributed

import (
	"encoding/json"
	"fmt"
)

// ZeroStage represents DeepSpeed ZeRO optimization stages.
type ZeroStage int

const (
	ZeroStage0 ZeroStage = 0 // Disabled
	ZeroStage1 ZeroStage = 1 // Optimizer State Partitioning
	ZeroStage2 ZeroStage = 2 // Gradient Partitioning
	ZeroStage3 ZeroStage = 3 // Parameter Partitioning
)

// DeepSpeedConfig helps generate deepspeed.json configuration.
type DeepSpeedConfig struct {
	TrainBatchSize       int         `json:"train_batch_size,omitempty"`
	TrainMicroBatchSize  int         `json:"train_micro_batch_size_per_gpu,omitempty"`
	GradientAccumulation int         `json:"gradient_accumulation_steps,omitempty"`
	ZeroOptimization     ZeroConfig  `json:"zero_optimization"`
	Optimizer            OptimConfig `json:"optimizer,omitempty"`
	FP16                 FPConfig    `json:"fp16,omitempty"`
	BF16                 FPConfig    `json:"bf16,omitempty"`
}

type ZeroConfig struct {
	Stage               int            `json:"stage"`
	OffloadOptimizer    *OffloadConfig `json:"offload_optimizer,omitempty"`
	OffloadParam        *OffloadConfig `json:"offload_param,omitempty"`
	OverlapComm         bool           `json:"overlap_comm,omitempty"`
	ContiguousGradients bool           `json:"contiguous_gradients,omitempty"`
}

type OffloadConfig struct {
	Device    string `json:"device"` // "cpu" or "nvme"
	PinMemory bool   `json:"pin_memory"`
}

type OptimConfig struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

type FPConfig struct {
	Enabled bool `json:"enabled"`
}

// NewDeepSpeedConfig creates a default DeepSpeed configuration.
func NewDeepSpeedConfig(stage ZeroStage) *DeepSpeedConfig {
	cfg := &DeepSpeedConfig{
		ZeroOptimization: ZeroConfig{
			Stage: int(stage),
		},
	}

	if stage == ZeroStage3 {
		cfg.ZeroOptimization.OffloadOptimizer = &OffloadConfig{Device: "cpu"}
		cfg.ZeroOptimization.OffloadParam = &OffloadConfig{Device: "cpu"}
	}

	return cfg
}

// WithFP16 enables float16 mixed precision.
func (c *DeepSpeedConfig) WithFP16() *DeepSpeedConfig {
	c.FP16 = FPConfig{Enabled: true}
	return c
}

// WithBF16 enables bfloat16 mixed precision.
func (c *DeepSpeedConfig) WithBF16() *DeepSpeedConfig {
	c.BF16 = FPConfig{Enabled: true}
	return c
}

// ToJSON returns the JSON string.
func (c *DeepSpeedConfig) ToJSON() (string, error) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// TorchElasticConfig configures torchrun.
type TorchElasticConfig struct {
	NNodes       int
	NProcPerNode int
	RDZVID       string
	RDZVBackend  string
	RDZVEpt      string
	MaxRestarts  int
}

// DefaultTorchElastic returns a single-node multi-gpu config.
func DefaultTorchElastic(gpus int) TorchElasticConfig {
	return TorchElasticConfig{
		NNodes:       1,
		NProcPerNode: gpus,
		RDZVBackend:  "c10d",
		RDZVEpt:      "localhost:29500",
		MaxRestarts:  3,
	}
}

// BuildCommand generates the torchrun command args.
func (c TorchElasticConfig) BuildCommand(script string, scriptArgs ...string) []string {
	args := []string{
		"-m", "torch.distributed.run",
		fmt.Sprintf("--nnodes=%d", c.NNodes),
		fmt.Sprintf("--nproc_per_node=%d", c.NProcPerNode),
		fmt.Sprintf("--rdzv_id=%s", c.RDZVID),
		fmt.Sprintf("--rdzv_backend=%s", c.RDZVBackend),
		fmt.Sprintf("--rdzv_endpoint=%s", c.RDZVEpt),
		fmt.Sprintf("--max_restarts=%d", c.MaxRestarts),
		script,
	}
	args = append(args, scriptArgs...)
	return args
}
