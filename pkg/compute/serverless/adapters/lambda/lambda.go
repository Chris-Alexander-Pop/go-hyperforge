// Package lambda provides an AWS Lambda adapter for serverless.ServerlessRuntime.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/compute/serverless/adapters/lambda"
//
//	runtime, err := lambda.New(lambda.Config{Region: "us-east-1"})
//	result, err := runtime.Invoke(ctx, serverless.InvokeOptions{FunctionName: "my-fn", Payload: data})
package lambda

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/compute/serverless"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds AWS Lambda configuration.
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // For LocalStack
}

// Runtime implements serverless.ServerlessRuntime for AWS Lambda.
type Runtime struct {
	client *lambda.Client
	config Config
}

// New creates a new Lambda runtime.
func New(cfg Config) (*Runtime, error) {
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

	clientOpts := []func(*lambda.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *lambda.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	return &Runtime{
		client: lambda.NewFromConfig(awsCfg, clientOpts...),
		config: cfg,
	}, nil
}

func (r *Runtime) CreateFunction(ctx context.Context, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	input := &lambda.CreateFunctionInput{
		FunctionName: aws.String(opts.Name),
		Runtime:      types.Runtime(opts.Runtime),
		Handler:      aws.String(opts.Handler),
		Role:         aws.String(opts.Role),
		Code: &types.FunctionCode{
			ZipFile: opts.Code,
		},
		MemorySize: aws.Int32(int32(opts.MemoryMB)),
		Timeout:    aws.Int32(int32(opts.TimeoutSeconds)),
	}

	if opts.Environment != nil {
		input.Environment = &types.Environment{
			Variables: opts.Environment,
		}
	}

	output, err := r.client.CreateFunction(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create function", err)
	}

	return &serverless.Function{
		Name:           *output.FunctionName,
		ARN:            *output.FunctionArn,
		Runtime:        serverless.Runtime(output.Runtime),
		Handler:        *output.Handler,
		MemoryMB:       int(*output.MemorySize),
		TimeoutSeconds: int(*output.Timeout),
		Version:        *output.Version,
		LastModified:   time.Now(),
		State:          string(output.State),
	}, nil
}

func (r *Runtime) GetFunction(ctx context.Context, name string) (*serverless.Function, error) {
	output, err := r.client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("function not found", err)
	}

	cfg := output.Configuration
	return &serverless.Function{
		Name:           *cfg.FunctionName,
		ARN:            *cfg.FunctionArn,
		Runtime:        serverless.Runtime(cfg.Runtime),
		Handler:        *cfg.Handler,
		MemoryMB:       int(*cfg.MemorySize),
		TimeoutSeconds: int(*cfg.Timeout),
		Version:        *cfg.Version,
		State:          string(cfg.State),
	}, nil
}

func (r *Runtime) ListFunctions(ctx context.Context) ([]*serverless.Function, error) {
	output, err := r.client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		return nil, pkgerrors.Internal("failed to list functions", err)
	}

	functions := make([]*serverless.Function, len(output.Functions))
	for i, fn := range output.Functions {
		functions[i] = &serverless.Function{
			Name:           *fn.FunctionName,
			ARN:            *fn.FunctionArn,
			Runtime:        serverless.Runtime(fn.Runtime),
			MemoryMB:       int(*fn.MemorySize),
			TimeoutSeconds: int(*fn.Timeout),
			Version:        *fn.Version,
		}
	}

	return functions, nil
}

func (r *Runtime) UpdateFunction(ctx context.Context, name string, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	// Update code if provided
	if len(opts.Code) > 0 {
		_, err := r.client.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(name),
			ZipFile:      opts.Code,
		})
		if err != nil {
			return nil, pkgerrors.Internal("failed to update function code", err)
		}
	}

	// Update configuration
	input := &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(name),
	}
	if opts.MemoryMB > 0 {
		input.MemorySize = aws.Int32(int32(opts.MemoryMB))
	}
	if opts.TimeoutSeconds > 0 {
		input.Timeout = aws.Int32(int32(opts.TimeoutSeconds))
	}
	if opts.Environment != nil {
		input.Environment = &types.Environment{Variables: opts.Environment}
	}

	output, err := r.client.UpdateFunctionConfiguration(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to update function", err)
	}

	return &serverless.Function{
		Name:           *output.FunctionName,
		ARN:            *output.FunctionArn,
		MemoryMB:       int(*output.MemorySize),
		TimeoutSeconds: int(*output.Timeout),
		Version:        *output.Version,
		LastModified:   time.Now(),
	}, nil
}

func (r *Runtime) DeleteFunction(ctx context.Context, name string) error {
	_, err := r.client.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete function", err)
	}
	return nil
}

func (r *Runtime) Invoke(ctx context.Context, opts serverless.InvokeOptions) (*serverless.InvokeResult, error) {
	invocationType := types.InvocationTypeRequestResponse
	switch opts.InvocationType {
	case serverless.InvocationAsync:
		invocationType = types.InvocationTypeEvent
	case serverless.InvocationDryRun:
		invocationType = types.InvocationTypeDryRun
	}

	input := &lambda.InvokeInput{
		FunctionName:   aws.String(opts.FunctionName),
		Payload:        opts.Payload,
		InvocationType: invocationType,
	}

	if opts.Qualifier != "" {
		input.Qualifier = aws.String(opts.Qualifier)
	}

	output, err := r.client.Invoke(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to invoke function", err)
	}

	result := &serverless.InvokeResult{
		StatusCode:      int(output.StatusCode),
		Payload:         output.Payload,
		ExecutedVersion: aws.ToString(output.ExecutedVersion),
	}

	if output.FunctionError != nil {
		result.FunctionError = *output.FunctionError
	}
	if output.LogResult != nil {
		result.LogResult = *output.LogResult
	}

	return result, nil
}

func (r *Runtime) InvokeSimple(ctx context.Context, name string, payload []byte) ([]byte, error) {
	result, err := r.Invoke(ctx, serverless.InvokeOptions{
		FunctionName:   name,
		Payload:        payload,
		InvocationType: serverless.InvocationSync,
	})
	if err != nil {
		return nil, err
	}
	if result.FunctionError != "" {
		var errResp struct{ ErrorMessage string }
		if err := json.Unmarshal(result.Payload, &errResp); err != nil {
			return nil, pkgerrors.Internal("failed to parse function error", err)
		}
		return nil, pkgerrors.Internal("function error: "+errResp.ErrorMessage, nil)
	}
	return result.Payload, nil
}

var _ serverless.ServerlessRuntime = (*Runtime)(nil)
