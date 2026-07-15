// Package azurefunctions provides a thin Azure Functions adapter scaffold for
// serverless.ServerlessRuntime. Management-plane CRUD returns Unimplemented;
// Invoke can target a configured HTTP trigger URL when set.
package azurefunctions

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/serverless"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config holds Azure Functions configuration.
type Config struct {
	SubscriptionID string `env:"SERVERLESS_AZURE_SUBSCRIPTION"`
	ResourceGroup  string `env:"SERVERLESS_AZURE_RESOURCE_GROUP"`
	// InvokeBaseURL is an optional HTTP trigger base (e.g. https://app.azurewebsites.net/api).
	InvokeBaseURL string `env:"SERVERLESS_AZURE_INVOKE_URL"`
	HTTPClient    *http.Client
}

// Runtime implements serverless.ServerlessRuntime for Azure Functions (scaffold).
type Runtime struct {
	config     Config
	httpClient *http.Client
}

// New creates an Azure Functions runtime scaffold.
func New(cfg Config) (*Runtime, error) {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Runtime{config: cfg, httpClient: hc}, nil
}

func unimplemented(op string) error {
	return pkgerrors.Unimplemented("azurefunctions."+op+" requires Azure Functions ARM SDK wiring", nil)
}

func (r *Runtime) CreateFunction(ctx context.Context, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	return nil, unimplemented("CreateFunction")
}
func (r *Runtime) GetFunction(ctx context.Context, name string) (*serverless.Function, error) {
	return nil, unimplemented("GetFunction")
}
func (r *Runtime) ListFunctions(ctx context.Context) ([]*serverless.Function, error) {
	return nil, unimplemented("ListFunctions")
}
func (r *Runtime) UpdateFunction(ctx context.Context, name string, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	return nil, unimplemented("UpdateFunction")
}
func (r *Runtime) DeleteFunction(ctx context.Context, name string) error {
	return unimplemented("DeleteFunction")
}

func (r *Runtime) Invoke(ctx context.Context, opts serverless.InvokeOptions) (*serverless.InvokeResult, error) {
	if r.config.InvokeBaseURL == "" {
		return nil, unimplemented("Invoke")
	}
	if opts.FunctionName == "" {
		return nil, pkgerrors.InvalidArgument("function_name is required", nil)
	}
	url := r.config.InvokeBaseURL + "/" + opts.FunctionName
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(opts.Payload))
	if err != nil {
		return nil, pkgerrors.Internal("failed to build invoke request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("azure function invoke failed", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	result := &serverless.InvokeResult{StatusCode: resp.StatusCode, Payload: body}
	if resp.StatusCode >= 400 {
		result.FunctionError = http.StatusText(resp.StatusCode)
	}
	return result, nil
}

func (r *Runtime) InvokeSimple(ctx context.Context, name string, payload []byte) ([]byte, error) {
	res, err := r.Invoke(ctx, serverless.InvokeOptions{FunctionName: name, Payload: payload})
	if err != nil {
		return nil, err
	}
	if res.FunctionError != "" {
		return res.Payload, serverless.ErrFunctionError
	}
	return res.Payload, nil
}

var _ serverless.ServerlessRuntime = (*Runtime)(nil)
