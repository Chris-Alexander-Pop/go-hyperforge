// Package apigateway provides a unified interface for API gateway management.
//
// Only the memory adapter ships today. AWS API Gateway / Apigee / Azure API
// Management / Kong driver constants are reserved placeholders — no cloud
// adapters yet.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/network/apigateway/adapters/memory"
//
//	gw := memory.New()
//	api, err := gw.CreateAPI(ctx, apigateway.CreateAPIOptions{Name: "my-api"})
package apigateway

import (
	"context"
	"time"
)

// Driver constants for API gateway backends.
// Only DriverMemory ships; cloud/gateway drivers are reserved placeholders.
const (
	DriverMemory = "memory"
	DriverAWS    = "aws"    // reserved — not implemented
	DriverApigee = "apigee" // reserved — not implemented
	DriverAzure  = "azure"  // reserved — not implemented
	DriverKong   = "kong"   // reserved — not implemented
)

// APIType represents the API type.
type APIType string

const (
	APITypeREST      APIType = "REST"
	APITypeHTTP      APIType = "HTTP"
	APITypeWebSocket APIType = "WEBSOCKET"
)

// Config holds configuration for API gateway management.
type Config struct {
	// Driver specifies the API gateway backend.
	Driver string `env:"APIGATEWAY_DRIVER" env-default:"memory"`

	// AWS specific
	AWSAccessKeyID     string `env:"APIGATEWAY_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"APIGATEWAY_AWS_SECRET_KEY"`
	AWSRegion          string `env:"APIGATEWAY_AWS_REGION" env-default:"us-east-1"`

	// GCP Apigee specific
	GCPProjectID string `env:"APIGATEWAY_GCP_PROJECT"`
	ApigeeOrg    string `env:"APIGEE_ORG"`

	// Azure specific
	AzureSubscriptionID string `env:"APIGATEWAY_AZURE_SUBSCRIPTION"`
	AzureResourceGroup  string `env:"APIGATEWAY_AZURE_RESOURCE_GROUP"`
}

// API represents an API configuration.
type API struct {
	// ID is the unique identifier.
	ID string

	// Name is the API name.
	Name string

	// Description is the API description.
	Description string

	// Type is the API type.
	Type APIType

	// Endpoint is the API invoke URL.
	Endpoint string

	// Version is the API version.
	Version string

	// Routes are the API routes.
	Routes []Route

	// Stages are deployment stages.
	Stages []Stage

	// Tags are key-value metadata.
	Tags map[string]string

	// CreatedAt is when the API was created.
	CreatedAt time.Time
}

// Route represents an API route.
type Route struct {
	// ID is the route identifier.
	ID string

	// Method is the HTTP method.
	Method string

	// Path is the route path.
	Path string

	// Integration is the backend integration.
	Integration Integration

	// Authorization is the auth type.
	Authorization string

	// AuthorizerID is the authorizer ID.
	AuthorizerID string
}

// Integration represents a backend integration.
type Integration struct {
	// Type is the integration type (HTTP, Lambda, etc.).
	Type string

	// URI is the backend URI.
	URI string

	// Method is the backend method.
	Method string

	// TimeoutMs is the timeout in milliseconds.
	TimeoutMs int
}

// Stage represents a deployment stage.
type Stage struct {
	// Name is the stage name.
	Name string

	// Description is the stage description.
	Description string

	// Variables are stage variables.
	Variables map[string]string

	// DeployedAt is when the stage was deployed.
	DeployedAt time.Time
}

// CreateAPIOptions configures API creation.
type CreateAPIOptions struct {
	// Name is the API name.
	Name string

	// Description is the API description.
	Description string

	// Type is the API type.
	Type APIType

	// Version is the API version.
	Version string

	// Tags are key-value metadata.
	Tags map[string]string
}

// APIGatewayManager defines the interface for API gateway management.
type APIGatewayManager interface {
	// CreateAPI creates a new API.
	CreateAPI(ctx context.Context, opts CreateAPIOptions) (*API, error)

	// GetAPI retrieves an API by ID.
	GetAPI(ctx context.Context, id string) (*API, error)

	// ListAPIs returns all APIs.
	ListAPIs(ctx context.Context) ([]*API, error)

	// DeleteAPI deletes an API.
	DeleteAPI(ctx context.Context, id string) error

	// AddRoute adds a route to an API.
	AddRoute(ctx context.Context, apiID string, route Route) (*Route, error)

	// RemoveRoute removes a route from an API.
	RemoveRoute(ctx context.Context, apiID, routeID string) error

	// Deploy deploys an API to a stage.
	Deploy(ctx context.Context, apiID, stageName string) (*Stage, error)

	// GetStage retrieves a stage.
	GetStage(ctx context.Context, apiID, stageName string) (*Stage, error)
}
