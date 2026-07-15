// Package memory provides an in-memory implementation of serverless.ServerlessRuntime.
package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute/serverless"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// HandlerFunc is a function that can be registered as a serverless handler.
type HandlerFunc func(ctx context.Context, payload []byte) ([]byte, error)

// Runtime implements an in-memory serverless runtime for testing.
type Runtime struct {
	mu        *concurrency.SmartRWMutex
	functions map[string]*serverless.Function
	handlers  map[string]HandlerFunc
	config    serverless.Config
}

// New creates a new in-memory serverless runtime.
func New() *Runtime {
	return &Runtime{
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "compute-serverless-memory",
		}),
		functions: make(map[string]*serverless.Function),
		handlers:  make(map[string]HandlerFunc),
		config:    serverless.Config{DefaultTimeout: 30 * time.Second, DefaultMemory: 128},
	}
}

// RegisterHandler registers a handler for a function name (for testing).
func (r *Runtime) RegisterHandler(name string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = handler
}

func (r *Runtime) CreateFunction(ctx context.Context, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.functions[opts.Name]; exists {
		return nil, serverless.ErrFunctionAlreadyExists
	}

	memory := opts.MemoryMB
	if memory <= 0 {
		memory = r.config.DefaultMemory
	}

	timeout := opts.TimeoutSeconds
	if timeout <= 0 {
		timeout = int(r.config.DefaultTimeout.Seconds())
	}

	fn := &serverless.Function{
		Name:           opts.Name,
		ARN:            "arn:aws:lambda:us-east-1:123456789012:function:" + opts.Name,
		Runtime:        opts.Runtime,
		Handler:        opts.Handler,
		MemoryMB:       memory,
		TimeoutSeconds: timeout,
		Environment:    opts.Environment,
		Version:        "$LATEST",
		LastModified:   time.Now(),
		State:          "Active",
	}

	r.functions[opts.Name] = fn

	if _, ok := r.handlers[opts.Name]; !ok {
		r.handlers[opts.Name] = func(ctx context.Context, payload []byte) ([]byte, error) {
			return payload, nil
		}
	}

	return fn, nil
}

func (r *Runtime) GetFunction(ctx context.Context, name string) (*serverless.Function, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, ok := r.functions[name]
	if !ok {
		return nil, serverless.ErrFunctionNotFound
	}

	return fn, nil
}

func (r *Runtime) ListFunctions(ctx context.Context) ([]*serverless.Function, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	functions := make([]*serverless.Function, 0, len(r.functions))
	for _, fn := range r.functions {
		functions = append(functions, fn)
	}

	return functions, nil
}

func (r *Runtime) UpdateFunction(ctx context.Context, name string, opts serverless.CreateFunctionOptions) (*serverless.Function, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	fn, ok := r.functions[name]
	if !ok {
		return nil, serverless.ErrFunctionNotFound
	}

	if opts.MemoryMB > 0 {
		fn.MemoryMB = opts.MemoryMB
	}
	if opts.TimeoutSeconds > 0 {
		fn.TimeoutSeconds = opts.TimeoutSeconds
	}
	if opts.Environment != nil {
		fn.Environment = opts.Environment
	}
	if opts.Handler != "" {
		fn.Handler = opts.Handler
	}
	fn.LastModified = time.Now()

	return fn, nil
}

func (r *Runtime) DeleteFunction(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.functions[name]; !ok {
		return serverless.ErrFunctionNotFound
	}

	delete(r.functions, name)
	delete(r.handlers, name)

	return nil
}

func (r *Runtime) Invoke(ctx context.Context, opts serverless.InvokeOptions) (*serverless.InvokeResult, error) {
	r.mu.RLock()
	fn, fnOk := r.functions[opts.FunctionName]
	handler, handlerOk := r.handlers[opts.FunctionName]
	r.mu.RUnlock()

	if !fnOk {
		return nil, serverless.ErrFunctionNotFound
	}

	result := &serverless.InvokeResult{
		ExecutedVersion: fn.Version,
	}

	if opts.InvocationType == serverless.InvocationDryRun {
		result.StatusCode = 204
		return result, nil
	}

	if opts.InvocationType == serverless.InvocationAsync {
		if handlerOk {
			go func() {
				execCtx, cancel := context.WithTimeout(context.Background(), time.Duration(fn.TimeoutSeconds)*time.Second)
				defer cancel()
				_, _ = handler(execCtx, opts.Payload)
			}()
		}
		result.StatusCode = 202
		return result, nil
	}

	if !handlerOk {
		result.StatusCode = 200
		result.Payload = opts.Payload
		return result, nil
	}

	execCtx := ctx
	if fn.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(fn.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	reqID := uuid.NewString()
	result.LogResult = "[" + reqID + "] START RequestId: " + reqID + "\n"

	payload, err := handler(execCtx, opts.Payload)
	if err != nil {
		result.StatusCode = 200
		result.FunctionError = "Unhandled"
		result.Payload = []byte(`{"errorMessage":"` + err.Error() + `"}`)
		result.LogResult += "[ERROR] " + err.Error() + "\n"
		return result, nil
	}

	result.StatusCode = 200
	result.Payload = payload
	result.LogResult += "[" + reqID + "] END RequestId: " + reqID + "\n"

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
		return nil, errors.Wrap(serverless.ErrFunctionError, "function error: "+result.FunctionError)
	}

	return result.Payload, nil
}
