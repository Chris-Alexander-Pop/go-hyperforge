// Package tuner provides hyperparameter optimization (HPO) utilities.
//
// Supports Grid Search, Random Search, and Bayesian Optimization (interface).
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training/tuner"
//
//	tuner := tuner.NewGridSearch(trainer, baseConfig, map[string][]inferface{}{"lr": {0.01, 0.001}})
//	bestJob, err := tuner.Run(ctx)
package tuner

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/training"
)

// Strategy defines the tuning strategy.
type Strategy string

const (
	StrategyGrid   Strategy = "grid"
	StrategyRandom Strategy = "random"
)

// SearchSpace defines the hyperparameters to tune.
type SearchSpace map[string][]interface{}

// Result holds the result of a single trial.
type Result struct {
	JobID           string
	Hyperparameters map[string]interface{}
	MetricValue     float64
	JobStatus       training.JobStatus
}

// Tuner manages the optimization process.
type Tuner struct {
	trainer    training.Trainer
	baseConfig training.JobConfig
	space      SearchSpace
	strategy   Strategy
	metricName string
	maximize   bool
	maxTrials  int
}

// NewGridSearch creates a grid search tuner.
func NewGridSearch(trainer training.Trainer, baseCfg training.JobConfig, space SearchSpace, metric string, maximize bool) *Tuner {
	// Calculate total combinations for grid search limit?
	// For now infinite/all combinations
	return &Tuner{
		trainer:    trainer,
		baseConfig: baseCfg,
		space:      space,
		strategy:   StrategyGrid,
		metricName: metric,
		maximize:   maximize,
		maxTrials:  1000,
	}
}

// NewRandomSearch creates a random search tuner.
func NewRandomSearch(trainer training.Trainer, baseCfg training.JobConfig, space SearchSpace, metric string, maxTrials int) *Tuner {
	return &Tuner{
		trainer:    trainer,
		baseConfig: baseCfg,
		space:      space,
		strategy:   StrategyRandom,
		metricName: metric,
		maximize:   true,
		maxTrials:  maxTrials,
	}
}

// Run executes the tuning process.
func (t *Tuner) Run(ctx context.Context) (*Result, error) {
	var combinations []map[string]interface{}

	if t.strategy == StrategyGrid {
		combinations = generateGrid(t.space)
	} else {
		combinations = generateRandom(t.space, t.maxTrials)
	}

	var bestResult *Result

	for i, params := range combinations {
		if i >= t.maxTrials {
			break
		}

		// Configure job
		jobConfig := t.baseConfig
		jobConfig.Name = fmt.Sprintf("%s-trial-%d", t.baseConfig.Name, i)

		// Copy base hyperparameters and overwrite with trial params
		newParams := make(map[string]interface{})
		for k, v := range t.baseConfig.Hyperparameters {
			newParams[k] = v
		}
		for k, v := range params {
			newParams[k] = v
		}
		jobConfig.Hyperparameters = newParams

		// Run job
		job, err := t.trainer.StartJob(ctx, jobConfig)
		if err != nil {
			// Log error and continue?
			continue
		}

		// Wait for completion (simple synchronous implementation for now)
		// in a real system this would be async/parallel
		finalJob := t.waitForJob(ctx, job.ID)

		val, ok := finalJob.Metrics[t.metricName]
		if !ok {
			// Metric missing in this trial
			continue
		}

		result := &Result{
			JobID:           finalJob.ID,
			Hyperparameters: params,
			MetricValue:     val,
			JobStatus:       finalJob.Status,
		}

		if bestResult == nil {
			bestResult = result
		} else {
			if t.maximize && val > bestResult.MetricValue {
				bestResult = result
			} else if !t.maximize && val < bestResult.MetricValue {
				bestResult = result
			}
		}
	}

	return bestResult, nil
}

func (t *Tuner) waitForJob(ctx context.Context, jobID string) *training.Job {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			job, err := t.trainer.GetJob(ctx, jobID)
			if err != nil {
				continue
			}
			if job.Status == training.StatusCompleted || job.Status == training.StatusFailed || job.Status == training.StatusStopped {
				return job
			}
		}
	}
}

// Helper to generate Cartesian product for grid search
func generateGrid(space SearchSpace) []map[string]interface{} {
	keys := make([]string, 0, len(space))
	for k := range space {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Deterministic order

	results := []map[string]interface{}{{}}

	for _, key := range keys {
		values := space[key]
		var nextResults []map[string]interface{}

		for _, res := range results {
			for _, val := range values {
				// Clone map
				newMap := make(map[string]interface{})
				for k, v := range res {
					newMap[k] = v
				}
				newMap[key] = val
				nextResults = append(nextResults, newMap)
			}
		}
		results = nextResults
	}
	return results
}

func generateRandom(space SearchSpace, n int) []map[string]interface{} {
	results := make([]map[string]interface{}, n)
	keys := make([]string, 0, len(space))
	for k := range space {
		keys = append(keys, k)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < n; i++ {
		params := make(map[string]interface{})
		for _, key := range keys {
			values := space[key]
			idx := rng.Intn(len(values))
			params[key] = values[idx]
		}
		results[i] = params
	}
	return results
}
