package serverless

import (
	"context"
	"time"
)

// Driver constants for serverless backends.
const (
	DriverMemory         = "memory"
	DriverLambda         = "lambda"
	DriverGCF            = "gcf"
	DriverAzureFunctions = "azure-functions" // HTTP Invoke; ARM CRUD Unimplemented
)

// Runtime represents the function runtime.
type Runtime string

const (
	RuntimeNodeJS18  Runtime = "nodejs18.x"
	RuntimeNodeJS20  Runtime = "nodejs20.x"
	RuntimePython39  Runtime = "python3.9"
	RuntimePython311 Runtime = "python3.11"
	RuntimeGo121     Runtime = "go1.x"
	RuntimeJava17    Runtime = "java17"
	RuntimeDotNet6   Runtime = "dotnet6"
)

// InvocationType represents how the function is invoked.
type InvocationType string

const (
	InvocationSync   InvocationType = "RequestResponse"
	InvocationAsync  InvocationType = "Event"
	InvocationDryRun InvocationType = "DryRun"
)

// Config holds configuration for serverless runtime.
type Config struct {
	// Driver specifies the serverless backend.
	Driver string `env:"SERVERLESS_DRIVER" env-default:"memory"`

	// AWS Lambda specific
	AWSAccessKeyID     string `env:"SERVERLESS_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"SERVERLESS_AWS_SECRET_KEY"`
	AWSRegion          string `env:"SERVERLESS_AWS_REGION" env-default:"us-east-1"`

	// GCP specific
	GCPProjectID string `env:"SERVERLESS_GCP_PROJECT"`
	GCPRegion    string `env:"SERVERLESS_GCP_REGION" env-default:"us-central1"`

	// Azure specific
	AzureSubscriptionID string `env:"SERVERLESS_AZURE_SUBSCRIPTION"`
	AzureResourceGroup  string `env:"SERVERLESS_AZURE_RESOURCE_GROUP"`

	// Common options
	DefaultTimeout time.Duration `env:"SERVERLESS_TIMEOUT" env-default:"30s"`
	DefaultMemory  int           `env:"SERVERLESS_MEMORY" env-default:"128"`
}

// Function represents a serverless function.
type Function struct {
	// Name is the function name.
	Name string

	// ARN is the function ARN/ID.
	ARN string

	// Runtime is the function runtime.
	Runtime Runtime

	// Handler is the function handler.
	Handler string

	// MemoryMB is the memory allocation.
	MemoryMB int

	// TimeoutSeconds is the function timeout.
	TimeoutSeconds int

	// Environment contains environment variables.
	Environment map[string]string

	// Version is the function version.
	Version string

	// LastModified is when the function was last modified.
	LastModified time.Time

	// State is the function state.
	State string
}

// CreateFunctionOptions configures function creation.
type CreateFunctionOptions struct {
	// Name is the function name.
	Name string

	// Runtime is the function runtime.
	Runtime Runtime

	// Handler is the function handler.
	Handler string

	// Code is the function code (zip bytes or S3 location).
	Code []byte

	// MemoryMB is the memory allocation.
	MemoryMB int

	// TimeoutSeconds is the function timeout.
	TimeoutSeconds int

	// Environment contains environment variables.
	Environment map[string]string

	// Role is the execution role ARN.
	Role string

	// VPCConfig configures VPC access.
	VPCConfig *VPCConfig

	// Tags are key-value metadata.
	Tags map[string]string
}

// VPCConfig configures VPC access for functions.
type VPCConfig struct {
	// SubnetIDs are the VPC subnets.
	SubnetIDs []string

	// SecurityGroupIDs are the security groups.
	SecurityGroupIDs []string
}

// InvokeOptions configures function invocation.
type InvokeOptions struct {
	// FunctionName is the function to invoke.
	FunctionName string

	// Payload is the input data.
	Payload []byte

	// InvocationType is sync, async, or dry-run.
	InvocationType InvocationType

	// LogType configures log return (None or Tail).
	LogType string

	// Qualifier is the function version/alias.
	Qualifier string
}

// InvokeResult contains the invocation result.
type InvokeResult struct {
	// StatusCode is the HTTP status code.
	StatusCode int

	// Payload is the response data.
	Payload []byte

	// FunctionError contains error type if function errored.
	FunctionError string

	// LogResult contains base64-encoded logs.
	LogResult string

	// ExecutedVersion is the version that was executed.
	ExecutedVersion string
}

// ServerlessRuntime defines the interface for serverless function management.
type ServerlessRuntime interface {
	// CreateFunction creates a new function.
	CreateFunction(ctx context.Context, opts CreateFunctionOptions) (*Function, error)

	// GetFunction retrieves a function by name.
	GetFunction(ctx context.Context, name string) (*Function, error)

	// ListFunctions returns all functions.
	ListFunctions(ctx context.Context) ([]*Function, error)

	// UpdateFunction updates a function's code or configuration.
	UpdateFunction(ctx context.Context, name string, opts CreateFunctionOptions) (*Function, error)

	// DeleteFunction deletes a function.
	DeleteFunction(ctx context.Context, name string) error

	// Invoke invokes a function.
	Invoke(ctx context.Context, opts InvokeOptions) (*InvokeResult, error)

	// InvokeSimple is a convenience method for simple sync invocations.
	InvokeSimple(ctx context.Context, name string, payload []byte) ([]byte, error)
}
