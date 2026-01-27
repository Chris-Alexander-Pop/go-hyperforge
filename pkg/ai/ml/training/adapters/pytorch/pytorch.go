// Package pytorch provides a PyTorch training adapter.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/training/adapters/pytorch"
//
//	trainer := pytorch.New(pytorch.Config{PythonPath: "/usr/bin/python3"})
//	job, err := trainer.StartJob(ctx, training.JobConfig{EntryPoint: "train.py"})
package pytorch

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/training"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

// Config holds PyTorch trainer configuration.
type Config struct {
	// PythonPath is the Python executable.
	PythonPath string

	// WorkDir is the working directory.
	WorkDir string

	// LogDir stores training logs.
	LogDir string

	// GPUDevices to use.
	GPUDevices string

	// Environment variables.
	Environment map[string]string

	// UseTorchrun uses torchrun for distributed training.
	UseTorchrun bool

	// NProc is the number of processes for distributed training.
	NProc int
}

// Trainer implements training.Trainer for PyTorch.
type Trainer struct {
	config   Config
	jobs     map[string]*jobState
	mu       sync.RWMutex
	callback training.Callback
}

type jobState struct {
	job     *training.Job
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	metrics []training.Metrics
	logs    []string
}

// New creates a new PyTorch trainer.
func New(cfg Config) *Trainer {
	if cfg.PythonPath == "" {
		cfg.PythonPath = "python3"
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "/tmp/pytorch-training"
	}
	if cfg.LogDir == "" {
		cfg.LogDir = filepath.Join(cfg.WorkDir, "logs")
	}
	if cfg.NProc == 0 {
		cfg.NProc = 1
	}

	return &Trainer{
		config:   cfg,
		jobs:     make(map[string]*jobState),
		callback: &training.NoOpCallback{},
	}
}

// SetCallback sets the training callback.
func (t *Trainer) SetCallback(cb training.Callback) {
	t.callback = cb
}

func (t *Trainer) StartJob(ctx context.Context, config training.JobConfig) (*training.Job, error) {
	jobID := uuid.NewString()
	if config.Name == "" {
		config.Name = "pytorch-job-" + jobID[:8]
	}

	jobDir := filepath.Join(t.config.WorkDir, jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		return nil, pkgerrors.Internal("failed to create job directory", err)
	}

	outputDir := config.OutputPath
	if outputDir == "" {
		outputDir = filepath.Join(jobDir, "output")
		os.MkdirAll(outputDir, 0755)
	}

	job := &training.Job{
		ID:         jobID,
		Name:       config.Name,
		Status:     training.StatusPending,
		CreatedAt:  time.Now(),
		OutputPath: outputDir,
		Config:     config,
		Metrics:    make(map[string]float64),
	}

	// Build command
	var cmd *exec.Cmd
	args := make([]string, 0)

	if t.config.UseTorchrun && t.config.NProc > 1 {
		// Use torchrun for distributed training
		args = append(args, "-m", "torch.distributed.run",
			fmt.Sprintf("--nproc_per_node=%d", t.config.NProc),
			config.EntryPoint,
		)
	} else {
		args = append(args, config.EntryPoint)
	}

	// Add hyperparameters
	for key, value := range config.Hyperparameters {
		args = append(args, fmt.Sprintf("--%s=%v", key, value))
	}
	args = append(args, fmt.Sprintf("--output_dir=%s", outputDir))

	jobCtx, cancel := context.WithCancel(ctx)
	cmd = exec.CommandContext(jobCtx, t.config.PythonPath, args...)
	cmd.Dir = jobDir

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range t.config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	if t.config.GPUDevices != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CUDA_VISIBLE_DEVICES=%s", t.config.GPUDevices))
	}

	state := &jobState{
		job:     job,
		cmd:     cmd,
		cancel:  cancel,
		metrics: make([]training.Metrics, 0),
		logs:    make([]string, 0),
	}

	t.mu.Lock()
	t.jobs[jobID] = state
	t.mu.Unlock()

	go t.runJob(state)

	return job, nil
}

func (t *Trainer) runJob(state *jobState) {
	job := state.job
	job.Status = training.StatusRunning
	now := time.Now()
	job.StartedAt = &now

	t.callback.OnJobStart(job)

	stdout, err := state.cmd.StdoutPipe()
	if err != nil {
		job.Status = training.StatusFailed
		job.Error = err.Error()
		t.callback.OnJobFailed(job, err)
		return
	}

	stderr, err := state.cmd.StderrPipe()
	if err != nil {
		job.Status = training.StatusFailed
		job.Error = err.Error()
		t.callback.OnJobFailed(job, err)
		return
	}

	if err := state.cmd.Start(); err != nil {
		job.Status = training.StatusFailed
		job.Error = err.Error()
		t.callback.OnJobFailed(job, err)
		return
	}

	// Capture output
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			state.logs = append(state.logs, line)
			t.parseMetrics(state, line)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			state.logs = append(state.logs, "[stderr] "+line)
		}
	}()

	err = state.cmd.Wait()
	completed := time.Now()
	job.CompletedAt = &completed

	if err != nil {
		job.Status = training.StatusFailed
		job.Error = err.Error()
		t.callback.OnJobFailed(job, err)
	} else {
		job.Status = training.StatusCompleted
		t.callback.OnJobComplete(job)
	}

	if job.StartedAt != nil {
		job.BillableSeconds = int(completed.Sub(*job.StartedAt).Seconds())
	}
}

func (t *Trainer) parseMetrics(state *jobState, line string) {
	var metrics map[string]interface{}
	if err := json.Unmarshal([]byte(line), &metrics); err == nil {
		m := training.Metrics{
			Timestamp: time.Now(),
			Custom:    make(map[string]float64),
		}
		for k, v := range metrics {
			if f, ok := v.(float64); ok {
				switch k {
				case "loss", "train_loss":
					m.Loss = f
				case "accuracy", "acc", "train_acc":
					m.Accuracy = f
				case "step", "global_step":
					m.Step = int64(f)
				case "epoch":
					m.Epoch = int(f)
				default:
					m.Custom[k] = f
				}
			}
		}
		state.metrics = append(state.metrics, m)
		t.callback.OnMetrics(state.job, m)
	}
}

func (t *Trainer) GetJob(ctx context.Context, jobID string) (*training.Job, error) {
	t.mu.RLock()
	state, ok := t.jobs[jobID]
	t.mu.RUnlock()

	if !ok {
		return nil, pkgerrors.NotFound("job not found", nil)
	}
	return state.job, nil
}

func (t *Trainer) ListJobs(ctx context.Context) ([]*training.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	jobs := make([]*training.Job, 0, len(t.jobs))
	for _, state := range t.jobs {
		jobs = append(jobs, state.job)
	}
	return jobs, nil
}

func (t *Trainer) StopJob(ctx context.Context, jobID string) error {
	t.mu.RLock()
	state, ok := t.jobs[jobID]
	t.mu.RUnlock()

	if !ok {
		return pkgerrors.NotFound("job not found", nil)
	}

	state.cancel()
	state.job.Status = training.StatusStopped
	return nil
}

func (t *Trainer) GetMetrics(ctx context.Context, jobID string) ([]training.Metrics, error) {
	t.mu.RLock()
	state, ok := t.jobs[jobID]
	t.mu.RUnlock()

	if !ok {
		return nil, pkgerrors.NotFound("job not found", nil)
	}
	return state.metrics, nil
}

func (t *Trainer) GetLogs(ctx context.Context, jobID string, tail int) ([]string, error) {
	t.mu.RLock()
	state, ok := t.jobs[jobID]
	t.mu.RUnlock()

	if !ok {
		return nil, pkgerrors.NotFound("job not found", nil)
	}

	logs := state.logs
	if tail > 0 && len(logs) > tail {
		logs = logs[len(logs)-tail:]
	}
	return logs, nil
}

func (t *Trainer) ListCheckpoints(ctx context.Context, jobID string) ([]training.Checkpoint, error) {
	t.mu.RLock()
	state, ok := t.jobs[jobID]
	t.mu.RUnlock()

	if !ok {
		return nil, pkgerrors.NotFound("job not found", nil)
	}

	checkpointDir := filepath.Join(state.job.OutputPath, "checkpoints")
	entries, err := os.ReadDir(checkpointDir)
	if err != nil {
		return []training.Checkpoint{}, nil
	}

	checkpoints := make([]training.Checkpoint, 0)
	for _, entry := range entries {
		info, _ := entry.Info()
		checkpoints = append(checkpoints, training.Checkpoint{
			ID:        entry.Name(),
			Path:      filepath.Join(checkpointDir, entry.Name()),
			CreatedAt: info.ModTime(),
		})
	}
	return checkpoints, nil
}

var _ training.Trainer = (*Trainer)(nil)
