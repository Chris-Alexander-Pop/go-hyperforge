// Package training provides abstractions for ML model training.
//
// Supports multiple training backends: TensorFlow, PyTorch, SageMaker, Vertex AI.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/training"
//
//	trainer := training.New(training.Config{Backend: "tensorflow"})
//	job, err := trainer.StartJob(ctx, training.JobConfig{Model: "resnet50", Dataset: "imagenet"})
package training

import (
	"context"
	"time"
)

// Backend identifies the training backend.
type Backend string

const (
	BackendTensorFlow Backend = "tensorflow"
	BackendPyTorch    Backend = "pytorch"
	BackendSageMaker  Backend = "sagemaker"
	BackendVertexAI   Backend = "vertexai"
	BackendAzureML    Backend = "azureml"
)

// JobStatus represents the state of a training job.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusStopped   JobStatus = "stopped"
)

// Config holds trainer configuration.
type Config struct {
	// Backend is the training framework.
	Backend Backend

	// Region for cloud-based training.
	Region string

	// Credentials for cloud authentication.
	Credentials map[string]string
}

// JobConfig configures a training job.
type JobConfig struct {
	// Name is the job identifier.
	Name string

	// Model is the model architecture or path.
	Model string

	// Dataset is the training data location.
	Dataset string

	// OutputPath is where to save artifacts.
	OutputPath string

	// Hyperparameters for training.
	Hyperparameters map[string]interface{}

	// InstanceType is the compute instance (for cloud).
	InstanceType string

	// InstanceCount is the number of instances.
	InstanceCount int

	// MaxRuntime is the maximum training duration.
	MaxRuntime time.Duration

	// Environment variables.
	Environment map[string]string

	// Framework version (e.g., "2.12.0" for TensorFlow).
	FrameworkVersion string

	// Script path or entry point.
	EntryPoint string

	// Tags for organization.
	Tags map[string]string
}

// Job represents a training job.
type Job struct {
	// ID is the unique job identifier.
	ID string

	// Name is the job name.
	Name string

	// Status is the current status.
	Status JobStatus

	// CreatedAt is when the job was created.
	CreatedAt time.Time

	// StartedAt is when training started.
	StartedAt *time.Time

	// CompletedAt is when training finished.
	CompletedAt *time.Time

	// Metrics contains training metrics.
	Metrics map[string]float64

	// OutputPath is where artifacts are stored.
	OutputPath string

	// Error message if failed.
	Error string

	// BillableSeconds is the charged compute time.
	BillableSeconds int

	// Config is the job configuration.
	Config JobConfig
}

// Metrics contains training metrics at a point in time.
type Metrics struct {
	Step      int64
	Epoch     int
	Loss      float64
	Accuracy  float64
	Custom    map[string]float64
	Timestamp time.Time
}

// Checkpoints represent saved model states.
type Checkpoint struct {
	ID        string
	Step      int64
	Epoch     int
	Path      string
	Metrics   *Metrics
	CreatedAt time.Time
}

// Trainer is the interface for ML training backends.
type Trainer interface {
	// StartJob creates and starts a training job.
	StartJob(ctx context.Context, config JobConfig) (*Job, error)

	// GetJob retrieves a job by ID.
	GetJob(ctx context.Context, jobID string) (*Job, error)

	// ListJobs returns all jobs.
	ListJobs(ctx context.Context) ([]*Job, error)

	// StopJob stops a running job.
	StopJob(ctx context.Context, jobID string) error

	// GetMetrics retrieves training metrics.
	GetMetrics(ctx context.Context, jobID string) ([]Metrics, error)

	// GetLogs retrieves training logs.
	GetLogs(ctx context.Context, jobID string, tail int) ([]string, error)

	// ListCheckpoints returns saved checkpoints.
	ListCheckpoints(ctx context.Context, jobID string) ([]Checkpoint, error)
}

// Callback is called during training events.
type Callback interface {
	OnJobStart(job *Job)
	OnJobComplete(job *Job)
	OnJobFailed(job *Job, err error)
	OnMetrics(job *Job, metrics Metrics)
	OnCheckpoint(job *Job, checkpoint Checkpoint)
}

// NoOpCallback is a callback that does nothing.
type NoOpCallback struct{}

func (c *NoOpCallback) OnJobStart(job *Job)                  {}
func (c *NoOpCallback) OnJobComplete(job *Job)               {}
func (c *NoOpCallback) OnJobFailed(job *Job, err error)      {}
func (c *NoOpCallback) OnMetrics(job *Job, metrics Metrics)  {}
func (c *NoOpCallback) OnCheckpoint(job *Job, cp Checkpoint) {}

// DataLoader configures data loading for training.
type DataLoader struct {
	// BatchSize is the training batch size.
	BatchSize int

	// Shuffle enables data shuffling.
	Shuffle bool

	// NumWorkers for parallel loading.
	NumWorkers int

	// PrefetchFactor for prefetching.
	PrefetchFactor int
}

// Optimizer configures the training optimizer.
type Optimizer struct {
	// Name is the optimizer (adam, sgd, etc.).
	Name string

	// LearningRate is the base learning rate.
	LearningRate float64

	// WeightDecay for regularization.
	WeightDecay float64

	// Momentum for SGD.
	Momentum float64

	// Beta1 for Adam.
	Beta1 float64

	// Beta2 for Adam.
	Beta2 float64
}

// LRScheduler configures learning rate scheduling.
type LRScheduler struct {
	// Type is the scheduler type.
	Type string // "step", "cosine", "linear", "exponential"

	// StepSize for step scheduler.
	StepSize int

	// Gamma decay factor.
	Gamma float64

	// Warmup steps.
	WarmupSteps int
}

// DistributedConfig configures distributed training.
type DistributedConfig struct {
	// Strategy is the distribution strategy.
	Strategy string // "data_parallel", "model_parallel", "horovod"

	// NumGPUs per node.
	NumGPUs int

	// NumNodes in the cluster.
	NumNodes int
}
