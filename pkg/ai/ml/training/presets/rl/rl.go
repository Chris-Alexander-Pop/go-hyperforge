// Package rl provides presets for Reinforcement Learning training.
//
// Supports PPO, DQN, SAC and environment configuration.
//
// Usage:
//
//	config := rl.NewPPOConfig("CartPole-v1")
package rl

import (
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training"
)

// Algorithm defines the RL algorithm.
type Algorithm string

const (
	AlgoPPO Algorithm = "PPO" // Proximal Policy Optimization
	AlgoDQN Algorithm = "DQN" // Deep Q-Network
	AlgoSAC Algorithm = "SAC" // Soft Actor-Critic
)

// Config configures an RL job.
type Config struct {
	Algorithm      Algorithm
	EnvironmentID  string // Gym environment ID
	TotalTimesteps int
	PolicyType     string // "MlpPolicy", "CnnPolicy"
	LearningRate   float64
	Gamma          float64 // Discount factor
	NumEnvs        int     // Parallel environments
	Seed           int
}

// NewPPOConfig creates a default PPO configuration.
func NewPPOConfig(envID string) *Config {
	return &Config{
		Algorithm:      AlgoPPO,
		EnvironmentID:  envID,
		TotalTimesteps: 1000000,
		PolicyType:     "MlpPolicy",
		LearningRate:   3e-4,
		Gamma:          0.99,
		NumEnvs:        4,
		Seed:           42,
	}
}

// ToJobConfig converts the preset to a JobConfig.
func (c *Config) ToJobConfig() training.JobConfig {
	hyperparams := map[string]interface{}{
		"algorithm":       string(c.Algorithm),
		"env_id":          c.EnvironmentID,
		"total_timesteps": c.TotalTimesteps,
		"policy_type":     c.PolicyType,
		"learning_rate":   c.LearningRate,
		"gamma":           c.Gamma,
		"n_envs":          c.NumEnvs,
		"seed":            c.Seed,
	}

	tags := map[string]string{
		"preset": "rl",
		"algo":   string(c.Algorithm),
		"env":    c.EnvironmentID,
	}

	return training.JobConfig{
		Name:            fmt.Sprintf("rl-%s-%s", c.Algorithm, c.EnvironmentID),
		Model:           c.PolicyType,
		Dataset:         "simulated",
		Hyperparameters: hyperparams,
		EntryPoint:      "train_rl.py",
		Tags:            tags,
	}
}
