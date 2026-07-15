// Package meta provides presets for Meta-Learning algorithms.
//
// Supports MAML (Model-Agnostic Meta-Learning), Reptile, and Prototypical Networks.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training/presets/meta"
//
//	config := meta.NewMAMLConfig("omniglot", 5, 1) // 5-way 1-shot
package meta

import (
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training"
)

// Algorithm defines the meta-learning algorithm.
type Algorithm string

const (
	AlgoMAML     Algorithm = "MAML"
	AlgoReptile  Algorithm = "REPTILE"
	AlgoProtoNet Algorithm = "PROTO_NET"
)

// Config configures a meta-learning job.
type Config struct {
	Algorithm     Algorithm
	Dataset       string
	Ways          int // N-way classification
	Shots         int // K-shot learning
	Query         int // Number of query examples
	InnerSteps    int // Inner loop gradient steps
	InnerLR       float64
	OuterLR       float64
	MetaBatchSize int // Number of tasks per outer step
	Iterations    int
}

// NewMAMLConfig creates a default MAML configuration.
func NewMAMLConfig(dataset string, ways, shots int) *Config {
	return &Config{
		Algorithm:     AlgoMAML,
		Dataset:       dataset,
		Ways:          ways,
		Shots:         shots,
		Query:         15,
		InnerSteps:    5,
		InnerLR:       0.01,
		OuterLR:       0.001,
		MetaBatchSize: 4,
		Iterations:    60000,
	}
}

// ToJobConfig converts the preset to a JobConfig.
func (c *Config) ToJobConfig() training.JobConfig {
	hyperparams := map[string]interface{}{
		"algorithm":       string(c.Algorithm),
		"n_way":           c.Ways,
		"k_shot":          c.Shots,
		"n_query":         c.Query,
		"inner_steps":     c.InnerSteps,
		"inner_lr":        c.InnerLR,
		"outer_lr":        c.OuterLR,
		"meta_batch_size": c.MetaBatchSize,
		"iterations":      c.Iterations,
	}

	tags := map[string]string{
		"preset": "meta-learning",
		"algo":   string(c.Algorithm),
	}

	description := fmt.Sprintf("%s-%dway-%dshot", c.Algorithm, c.Ways, c.Shots)

	return training.JobConfig{
		Name:            description,
		Model:           "conv4", // Default backbone for few-shot
		Dataset:         c.Dataset,
		Hyperparameters: hyperparams,
		EntryPoint:      "train_meta.py",
		Tags:            tags,
	}
}
