// Package llm provides presets for Large Language Model training.
//
// Supports Fine-tuning (LoRA, QLoRA), Pre-training configurations, and prompt datasets.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training/presets/llm"
//
//	config := llm.NewFineTuneConfig("llama-3-8b", "dataset_path")
//	config.UseQLoRA()
//	jobConfig := config.ToJobConfig()
package llm

import (
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training/distributed"
)

// TaskType defines the LLM task.
type TaskType string

const (
	TaskCausalLM       TaskType = "CAUSAL_LM"    // Text completion / Chat
	TaskSeq2Seq        TaskType = "SEQ_2_SEQ_LM" // Translation / Summarization
	TaskClassification TaskType = "CLASSIFICATION"
)

// FineTuneConfig configures LLM fine-tuning.
type FineTuneConfig struct {
	BaseModel    string
	DatasetPath  string
	Task         TaskType
	Epochs       int
	LearningRate float64
	BatchSize    int
	MaxLength    int

	// PEFT configuration
	UseLoRA     bool
	LoRARank    int
	LoRAAlpha   int
	LoRADropout float64

	// Quantization
	Use4Bit bool // QLoRA
	Use8Bit bool

	// Distributed
	NumGPUs   int
	DeepSpeed bool
}

// NewFineTuneConfig creates a default fine-tuning configuration.
func NewFineTuneConfig(model, dataset string) *FineTuneConfig {
	return &FineTuneConfig{
		BaseModel:    model,
		DatasetPath:  dataset,
		Task:         TaskCausalLM,
		Epochs:       3,
		LearningRate: 2e-4,
		BatchSize:    4,
		MaxLength:    2048,
		NumGPUs:      1,
	}
}

// UseLoRA enables Low-Rank Adaptation.
func (c *FineTuneConfig) WithLoRA(r, alpha int) *FineTuneConfig {
	c.UseLoRA = true
	c.LoRARank = r
	c.LoRAAlpha = alpha
	c.LoRADropout = 0.05
	return c
}

// UseQLoRA enables 4-bit quantization with LoRA.
func (c *FineTuneConfig) UseQLoRA() *FineTuneConfig {
	c.Use4Bit = true
	c.UseLoRA = true
	if c.LoRARank == 0 {
		c.LoRARank = 64
		c.LoRAAlpha = 16
	}
	return c
}

// ToJobConfig converts the preset to a generic JobConfig.
func (c *FineTuneConfig) ToJobConfig() training.JobConfig {
	hyperparams := map[string]interface{}{
		"model_name_or_path":          c.BaseModel,
		"dataset_name":                c.DatasetPath,
		"num_train_epochs":            c.Epochs,
		"learning_rate":               c.LearningRate,
		"per_device_train_batch_size": c.BatchSize,
		"max_seq_length":              c.MaxLength,
	}

	tags := map[string]string{
		"preset": "llm-finetune",
		"model":  c.BaseModel,
	}

	if c.UseLoRA {
		hyperparams["use_peft"] = true
		hyperparams["lora_r"] = c.LoRARank
		hyperparams["lora_alpha"] = c.LoRAAlpha
		hyperparams["lora_dropout"] = c.LoRADropout
		tags["method"] = "lora"
	}

	if c.Use4Bit {
		hyperparams["load_in_4bit"] = true
		tags["quantization"] = "4bit"
	} else if c.Use8Bit {
		hyperparams["load_in_8bit"] = true
		tags["quantization"] = "8bit"
	}

	// NOTE:In a real system, the EntryPoint would be a standard training script
	// exposed by the library, e.g., "scripts/train_llm.py"
	entryPoint := "train_llm.py"

	cfg := training.JobConfig{
		Name:             fmt.Sprintf("llm-ft-%s", time.Now().Format("20060102-1504")),
		Model:            c.BaseModel,
		Dataset:          c.DatasetPath,
		Hyperparameters:  hyperparams,
		InstanceCount:    1, // Usually single node for simple FT
		EntryPoint:       entryPoint,
		Tags:             tags,
		FrameworkVersion: "transformers-4.35",
	}

	// Handle Distributed
	if c.NumGPUs > 1 {
		distCfg := distributed.DefaultTorchElastic(c.NumGPUs)
		cfg.Environment = map[string]string{
			"NUM_GPUS":       fmt.Sprintf("%d", c.NumGPUs),
			"TORCH_ELASTIC":  "1",
			"RDZV_ENDPOINT":  distCfg.RDZVEpt,
			"NPROC_PER_NODE": fmt.Sprintf("%d", distCfg.NProcPerNode),
		}
		if c.DeepSpeed {
			ds := distributed.NewDeepSpeedConfig(distributed.ZeroStage2)
			if c.Use4Bit || c.Use8Bit {
				// ZeRO-2 usually works with quantization, ZeRO-3 is harder
				ds.WithBF16()
			}
			dsJSON, _ := ds.ToJSON()
			hyperparams["deepspeed_config"] = dsJSON
			cfg.InstanceType = "multi-gpu" // generic indicator
		}
		// In a real implementation, the runner would handle the torchrun command construction
		// based on these flags or InstanceCount
		cfg.Environment = map[string]string{
			"NUM_GPUS": fmt.Sprintf("%d", c.NumGPUs),
		}
	}

	return cfg
}

// RLHFConfig configures Reinforcement Learning from Human Feedback.
type RLHFConfig struct {
	SFTModel    string
	RewardModel string
	DatasetPath string
	PPOEpochs   int
}
