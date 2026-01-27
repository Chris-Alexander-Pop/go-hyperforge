// Package sagemaker provides an AWS SageMaker training adapter.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/sagemaker"
//
//	trainer, err := sagemaker.New(sagemaker.Config{Region: "us-east-1"})
//	job, err := trainer.StartJob(ctx, training.JobConfig{...})
package sagemaker

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/training"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds SageMaker configuration.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	RoleARN         string
	S3OutputPath    string
}

// Trainer implements training.Trainer for AWS SageMaker.
type Trainer struct {
	client *sagemaker.Client
	config Config
}

// New creates a new SageMaker trainer.
func New(cfg Config) (*Trainer, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to load AWS config", err)
	}

	return &Trainer{
		client: sagemaker.NewFromConfig(awsCfg),
		config: cfg,
	}, nil
}

func (t *Trainer) StartJob(ctx context.Context, cfg training.JobConfig) (*training.Job, error) {
	name := cfg.Name
	if name == "" {
		name = fmt.Sprintf("training-%d", time.Now().Unix())
	}

	instanceType := types.TrainingInstanceTypeMlM5Large
	if cfg.InstanceType != "" {
		instanceType = types.TrainingInstanceType(cfg.InstanceType)
	}

	instanceCount := int32(1)
	if cfg.InstanceCount > 0 {
		instanceCount = int32(cfg.InstanceCount)
	}

	hyperparams := make(map[string]string)
	for k, v := range cfg.Hyperparameters {
		hyperparams[k] = fmt.Sprintf("%v", v)
	}

	outputPath := cfg.OutputPath
	if outputPath == "" {
		outputPath = t.config.S3OutputPath + "/" + name
	}

	input := &sagemaker.CreateTrainingJobInput{
		TrainingJobName: aws.String(name),
		RoleArn:         aws.String(t.config.RoleARN),
		AlgorithmSpecification: &types.AlgorithmSpecification{
			TrainingImage:     aws.String(cfg.Model),
			TrainingInputMode: types.TrainingInputModeFile,
		},
		ResourceConfig: &types.ResourceConfig{
			InstanceType:   instanceType,
			InstanceCount:  aws.Int32(instanceCount),
			VolumeSizeInGB: aws.Int32(50),
		},
		StoppingCondition: &types.StoppingCondition{
			MaxRuntimeInSeconds: aws.Int32(int32(cfg.MaxRuntime.Seconds())),
		},
		OutputDataConfig: &types.OutputDataConfig{
			S3OutputPath: aws.String(outputPath),
		},
		HyperParameters: hyperparams,
		InputDataConfig: []types.Channel{
			{
				ChannelName: aws.String("training"),
				DataSource: &types.DataSource{
					S3DataSource: &types.S3DataSource{
						S3DataType: types.S3DataTypeS3Prefix,
						S3Uri:      aws.String(cfg.Dataset),
					},
				},
			},
		},
	}

	if int32(cfg.MaxRuntime.Seconds()) == 0 {
		input.StoppingCondition.MaxRuntimeInSeconds = aws.Int32(86400) // 24 hours default
	}

	_, err := t.client.CreateTrainingJob(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create training job", err)
	}

	return &training.Job{
		ID:         name,
		Name:       name,
		Status:     training.StatusPending,
		CreatedAt:  time.Now(),
		OutputPath: outputPath,
		Config:     cfg,
		Metrics:    make(map[string]float64),
	}, nil
}

func (t *Trainer) GetJob(ctx context.Context, jobID string) (*training.Job, error) {
	output, err := t.client.DescribeTrainingJob(ctx, &sagemaker.DescribeTrainingJobInput{
		TrainingJobName: aws.String(jobID),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("job not found", err)
	}

	status := mapStatus(output.TrainingJobStatus)

	job := &training.Job{
		ID:         *output.TrainingJobName,
		Name:       *output.TrainingJobName,
		Status:     status,
		CreatedAt:  *output.CreationTime,
		OutputPath: *output.OutputDataConfig.S3OutputPath,
		Metrics:    make(map[string]float64),
	}

	if output.TrainingStartTime != nil {
		job.StartedAt = output.TrainingStartTime
	}
	if output.TrainingEndTime != nil {
		job.CompletedAt = output.TrainingEndTime
	}
	if output.BillableTimeInSeconds != nil {
		job.BillableSeconds = int(*output.BillableTimeInSeconds)
	}
	if output.FailureReason != nil {
		job.Error = *output.FailureReason
	}

	// Extract final metrics
	for _, metric := range output.FinalMetricDataList {
		job.Metrics[*metric.MetricName] = float64(*metric.Value)
	}

	return job, nil
}

func mapStatus(s types.TrainingJobStatus) training.JobStatus {
	switch s {
	case types.TrainingJobStatusInProgress:
		return training.StatusRunning
	case types.TrainingJobStatusCompleted:
		return training.StatusCompleted
	case types.TrainingJobStatusFailed:
		return training.StatusFailed
	case types.TrainingJobStatusStopping, types.TrainingJobStatusStopped:
		return training.StatusStopped
	default:
		return training.StatusPending
	}
}

func (t *Trainer) ListJobs(ctx context.Context) ([]*training.Job, error) {
	output, err := t.client.ListTrainingJobs(ctx, &sagemaker.ListTrainingJobsInput{
		MaxResults: aws.Int32(100),
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list jobs", err)
	}

	jobs := make([]*training.Job, len(output.TrainingJobSummaries))
	for i, summary := range output.TrainingJobSummaries {
		jobs[i] = &training.Job{
			ID:        *summary.TrainingJobName,
			Name:      *summary.TrainingJobName,
			Status:    mapStatus(summary.TrainingJobStatus),
			CreatedAt: *summary.CreationTime,
		}
	}

	return jobs, nil
}

func (t *Trainer) StopJob(ctx context.Context, jobID string) error {
	_, err := t.client.StopTrainingJob(ctx, &sagemaker.StopTrainingJobInput{
		TrainingJobName: aws.String(jobID),
	})
	if err != nil {
		return pkgerrors.Internal("failed to stop job", err)
	}
	return nil
}

func (t *Trainer) GetMetrics(ctx context.Context, jobID string) ([]training.Metrics, error) {
	job, err := t.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	metrics := training.Metrics{
		Custom:    job.Metrics,
		Timestamp: time.Now(),
	}

	return []training.Metrics{metrics}, nil
}

func (t *Trainer) GetLogs(ctx context.Context, jobID string, tail int) ([]string, error) {
	// SageMaker logs go to CloudWatch Logs
	return []string{"Use CloudWatch Logs for SageMaker training logs"}, nil
}

func (t *Trainer) ListCheckpoints(ctx context.Context, jobID string) ([]training.Checkpoint, error) {
	// SageMaker checkpoints are in S3
	return []training.Checkpoint{}, nil
}

var _ training.Trainer = (*Trainer)(nil)
