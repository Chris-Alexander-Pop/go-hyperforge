// Package gcf provides a Google Cloud Functions adapter for serverless.ServerlessRuntime.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/compute/serverless/adapters/gcf"
//
//	runtime, err := gcf.New(gcf.Config{ProjectID: "my-project", Region: "us-central1"})
//	result, err := runtime.InvokeSimple(ctx, "my-function", payload)
package gcf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	functions "cloud.google.com/go/functions/apiv2"
	"cloud.google.com/go/functions/apiv2/functionspb"
	"github.com/chris-alexander-pop/system-design-library/pkg/compute/serverless"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Config holds GCF configuration.
type Config struct {
	ProjectID       string
	Region          string
	CredentialsFile string
	CredentialsJSON []byte
}

// Runtime implements serverless.ServerlessRuntime for GCF.
type Runtime struct {
	client     *functions.FunctionClient
	httpClient *http.Client
	config     Config
}

// New creates a new GCF runtime.
func New(cfg Config) (*Runtime, error) {
	ctx := context.Background()

	opts := []option.ClientOption{}
	if cfg.CredentialsFile != "" {
		b, err := os.ReadFile(cfg.CredentialsFile)
		if err != nil {
			return nil, pkgerrors.Internal("failed to read credentials file", err)
		}
		creds, err := google.CredentialsFromJSON(ctx, b, functions.DefaultAuthScopes()...)
		if err != nil {
			return nil, pkgerrors.Internal("failed to parse credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	if len(cfg.CredentialsJSON) > 0 {
		creds, err := google.CredentialsFromJSON(ctx, cfg.CredentialsJSON, functions.DefaultAuthScopes()...)
		if err != nil {
			return nil, pkgerrors.Internal("failed to parse credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	client, err := functions.NewFunctionRESTClient(ctx, opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create GCF client", err)
	}

	return &Runtime{
		client:     client,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		config:     cfg,
	}, nil
}

// Close closes the GCF client.
func (r *Runtime) Close() {
	if r.client != nil {
		r.client.Close()
	}
}

func (r *Runtime) functionName(name string) string {
	return fmt.Sprintf("projects/%s/locations/%s/functions/%s", r.config.ProjectID, r.config.Region, name)
}

func (r *Runtime) CreateFunction(ctx context.Context, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	fn := &functionspb.Function{
		Name:        r.functionName(opts.Name),
		Environment: functionspb.Environment_GEN_2,
		BuildConfig: &functionspb.BuildConfig{
			Runtime:    string(opts.Runtime),
			EntryPoint: opts.Handler,
		},
		ServiceConfig: &functionspb.ServiceConfig{
			AvailableMemory:      fmt.Sprintf("%dM", opts.MemoryMB),
			TimeoutSeconds:       int32(opts.TimeoutSeconds),
			EnvironmentVariables: opts.Environment,
		},
	}

	op, err := r.client.CreateFunction(ctx, &functionspb.CreateFunctionRequest{
		Parent:     fmt.Sprintf("projects/%s/locations/%s", r.config.ProjectID, r.config.Region),
		FunctionId: opts.Name,
		Function:   fn,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to create function", err)
	}

	result, err := op.Wait(ctx)
	if err != nil {
		return nil, pkgerrors.Internal("function creation failed", err)
	}

	return &serverless.Function{
		Name:         opts.Name,
		ARN:          result.Name,
		Runtime:      opts.Runtime,
		Handler:      opts.Handler,
		MemoryMB:     opts.MemoryMB,
		LastModified: time.Now(),
		State:        string(result.State),
	}, nil
}

func (r *Runtime) GetFunction(ctx context.Context, name string) (*serverless.Function, error) {
	fn, err := r.client.GetFunction(ctx, &functionspb.GetFunctionRequest{
		Name: r.functionName(name),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("function not found", err)
	}

	return &serverless.Function{
		Name:    name,
		ARN:     fn.Name,
		Runtime: serverless.Runtime(fn.BuildConfig.GetRuntime()),
		Handler: fn.BuildConfig.GetEntryPoint(),
		State:   string(fn.State),
	}, nil
}

func (r *Runtime) ListFunctions(ctx context.Context) ([]*serverless.Function, error) {
	it := r.client.ListFunctions(ctx, &functionspb.ListFunctionsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", r.config.ProjectID, r.config.Region),
	})

	var fns []*serverless.Function
	for {
		fn, err := it.Next()
		if err != nil {
			break
		}
		fns = append(fns, &serverless.Function{
			Name:    fn.Name,
			ARN:     fn.Name,
			Runtime: serverless.Runtime(fn.BuildConfig.GetRuntime()),
			State:   string(fn.State),
		})
	}

	return fns, nil
}

func (r *Runtime) UpdateFunction(ctx context.Context, name string, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	fn, err := r.GetFunction(ctx, name)
	if err != nil {
		return nil, err
	}

	update := &functionspb.Function{
		Name: r.functionName(name),
		ServiceConfig: &functionspb.ServiceConfig{
			EnvironmentVariables: opts.Environment,
		},
	}

	if opts.MemoryMB > 0 {
		update.ServiceConfig.AvailableMemory = fmt.Sprintf("%dM", opts.MemoryMB)
	}
	if opts.TimeoutSeconds > 0 {
		update.ServiceConfig.TimeoutSeconds = int32(opts.TimeoutSeconds)
	}

	op, err := r.client.UpdateFunction(ctx, &functionspb.UpdateFunctionRequest{
		Function: update,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to update function", err)
	}

	_, err = op.Wait(ctx)
	if err != nil {
		return nil, pkgerrors.Internal("function update failed", err)
	}

	fn.LastModified = time.Now()
	return fn, nil
}

func (r *Runtime) DeleteFunction(ctx context.Context, name string) error {
	op, err := r.client.DeleteFunction(ctx, &functionspb.DeleteFunctionRequest{
		Name: r.functionName(name),
	})
	if err != nil {
		return pkgerrors.Internal("failed to delete function", err)
	}

	if err := op.Wait(ctx); err != nil {
		return pkgerrors.Internal("function deletion failed", err)
	}

	return nil
}

func (r *Runtime) Invoke(ctx context.Context, opts serverless.InvokeOptions) (*serverless.InvokeResult, error) {
	// Get the function URL
	fn, err := r.client.GetFunction(ctx, &functionspb.GetFunctionRequest{
		Name: r.functionName(opts.FunctionName),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("function not found", err)
	}

	url := fn.ServiceConfig.GetUri()
	if url == "" {
		return nil, pkgerrors.Internal("function URL not available", nil)
	}

	// Make HTTP request to function URL
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(opts.Payload))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("failed to invoke function", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	result := &serverless.InvokeResult{
		StatusCode: resp.StatusCode,
		Payload:    body,
	}

	if resp.StatusCode >= 400 {
		result.FunctionError = "Unhandled"
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
		var errResp struct{ Error string }
		if err := json.Unmarshal(result.Payload, &errResp); err != nil {
			return nil, pkgerrors.Internal("failed to parse function error", err)
		}
		return nil, pkgerrors.Internal("function error: "+errResp.Error, nil)
	}
	return result.Payload, nil
}

var _ serverless.ServerlessRuntime = (*Runtime)(nil)
