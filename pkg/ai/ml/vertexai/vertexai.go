// Package vertexai provides a GCP Vertex AI training adapter.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/vertexai"
//
//	trainer, err := vertexai.New(vertexai.Config{ProjectID: "my-project", Region: "us-central1"})
//	job, err := trainer.StartJob(ctx, training.JobConfig{...})
package vertexai

import (
	"context"
	"fmt"
	"os"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/training"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

// Config holds Vertex AI configuration.
type Config struct {
	ProjectID       string
	Region          string
	CredentialsFile string
	CredentialsJSON []byte
	StagingBucket   string
}

// Trainer implements training.Trainer for Vertex AI.
type Trainer struct {
	client *aiplatform.JobClient
	config Config
	parent string
}

// New creates a new Vertex AI trainer.
func New(cfg Config) (*Trainer, error) {
	ctx := context.Background()

	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		b, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, pkgerrors.Internal("failed to read credentials file", err)
		}
		creds, err := google.CredentialsFromJSON(ctx, b, aiplatform.DefaultAuthScopes()...)
		if err != nil {
			return nil, pkgerrors.Internal("failed to parse credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	if len(cfg.CredentialsJSON) > 0 {
		creds, err := google.CredentialsFromJSON(ctx, cfg.CredentialsJSON, aiplatform.DefaultAuthScopes()...)
		if err != nil {
			return nil, pkgerrors.Internal("failed to parse credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", cfg.Region)
	opts = append(opts, option.WithEndpoint(endpoint))

	client, err := aiplatform.NewJobClient(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create Vertex AI client", err)
	}

	return &Trainer{
		client: client,
		config: cfg,
		parent: fmt.Sprintf("projects/%s/locations/%s", cfg.ProjectID, cfg.Region),
	}, nil
}

// Close closes the client.
func (t *Trainer) Close() {
	if t.client != nil {
		t.client.Close()
	}
}

func (t *Trainer) StartJob(ctx context.Context, cfg training.JobConfig) (*training.Job, error) {
	name := cfg.Name
	if name == "" {
		name = fmt.Sprintf("training-%d", time.Now().Unix())
	}

	// Convert hyperparameters to struct
	hyperparams, _ := structpb.NewStruct(cfg.Hyperparameters)

	machineType := "n1-standard-4"
	if cfg.InstanceType != "" {
		machineType = cfg.InstanceType
	}

	replicaCount := int32(1)
	if cfg.InstanceCount > 0 {
		replicaCount = int32(cfg.InstanceCount)
	}

	customJob := &aiplatformpb.CustomJob{
		DisplayName: name,
		JobSpec: &aiplatformpb.CustomJobSpec{
			WorkerPoolSpecs: []*aiplatformpb.WorkerPoolSpec{
				{
					ReplicaCount: int64(replicaCount),
					MachineSpec: &aiplatformpb.MachineSpec{
						MachineType: machineType,
					},
					Task: &aiplatformpb.WorkerPoolSpec_ContainerSpec{
						ContainerSpec: &aiplatformpb.ContainerSpec{
							ImageUri: cfg.Model,
							Command:  []string{cfg.EntryPoint},
							Args:     buildArgs(cfg.Hyperparameters),
						},
					},
				},
			},
			BaseOutputDirectory: &aiplatformpb.GcsDestination{
				OutputUriPrefix: cfg.OutputPath,
			},
		},
	}

	_ = hyperparams // Available for future use

	op, err := t.client.CreateCustomJob(ctx, &aiplatformpb.CreateCustomJobRequest{
		Parent:    t.parent,
		CustomJob: customJob,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create training job", err)
	}

	return &training.Job{
		ID:         op.Name,
		Name:       name,
		Status:     training.StatusPending,
		CreatedAt:  time.Now(),
		OutputPath: cfg.OutputPath,
		Config:     cfg,
		Metrics:    make(map[string]float64),
	}, nil
}

func buildArgs(hyperparams map[string]interface{}) []string {
	args := make([]string, 0)
	for k, v := range hyperparams {
		args = append(args, fmt.Sprintf("--%s=%v", k, v))
	}
	return args
}

func (t *Trainer) GetJob(ctx context.Context, jobID string) (*training.Job, error) {
	output, err := t.client.GetCustomJob(ctx, &aiplatformpb.GetCustomJobRequest{
		Name: jobID,
	})
	if err != nil {
		return nil, pkgerrors.NotFound("job not found", err)
	}

	status := mapVertexStatus(output.State)

	job := &training.Job{
		ID:        output.Name,
		Name:      output.DisplayName,
		Status:    status,
		CreatedAt: output.CreateTime.AsTime(),
		Metrics:   make(map[string]float64),
	}

	if output.StartTime != nil {
		t := output.StartTime.AsTime()
		job.StartedAt = &t
	}
	if output.EndTime != nil {
		t := output.EndTime.AsTime()
		job.CompletedAt = &t
	}
	if output.Error != nil {
		job.Error = output.Error.Message
	}

	return job, nil
}

func mapVertexStatus(s aiplatformpb.JobState) training.JobStatus {
	switch s {
	case aiplatformpb.JobState_JOB_STATE_RUNNING:
		return training.StatusRunning
	case aiplatformpb.JobState_JOB_STATE_SUCCEEDED:
		return training.StatusCompleted
	case aiplatformpb.JobState_JOB_STATE_FAILED:
		return training.StatusFailed
	case aiplatformpb.JobState_JOB_STATE_CANCELLED:
		return training.StatusStopped
	default:
		return training.StatusPending
	}
}

func (t *Trainer) ListJobs(ctx context.Context) ([]*training.Job, error) {
	it := t.client.ListCustomJobs(ctx, &aiplatformpb.ListCustomJobsRequest{
		Parent: t.parent,
	})

	var jobs []*training.Job
	for {
		resp, err := it.Next()
		if err != nil {
			break
		}
		jobs = append(jobs, &training.Job{
			ID:        resp.Name,
			Name:      resp.DisplayName,
			Status:    mapVertexStatus(resp.State),
			CreatedAt: resp.CreateTime.AsTime(),
		})
	}

	return jobs, nil
}

func (t *Trainer) StopJob(ctx context.Context, jobID string) error {
	err := t.client.CancelCustomJob(ctx, &aiplatformpb.CancelCustomJobRequest{
		Name: jobID,
	})
	if err != nil {
		return pkgerrors.Internal("failed to cancel job", err)
	}
	return nil
}

func (t *Trainer) GetMetrics(ctx context.Context, jobID string) ([]training.Metrics, error) {
	return []training.Metrics{}, nil
}

func (t *Trainer) GetLogs(ctx context.Context, jobID string, tail int) ([]string, error) {
	return []string{"Use Cloud Logging for Vertex AI training logs"}, nil
}

func (t *Trainer) ListCheckpoints(ctx context.Context, jobID string) ([]training.Checkpoint, error) {
	return []training.Checkpoint{}, nil
}

var _ training.Trainer = (*Trainer)(nil)
