// Package logicapps provides an Azure Logic Apps adapter for workflow.WorkflowEngine.
//
// Azure Logic Apps enables serverless workflow automation and integration.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/logicapps"
//
//	engine, err := logicapps.New(logicapps.Config{SubscriptionID: "...", ResourceGroup: "..."})
//	exec, err := engine.Start(ctx, workflow.StartOptions{WorkflowID: "my-logic-app", Input: data})
//	defer engine.Close()
package logicapps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"github.com/google/uuid"
)

// Config holds Azure Logic Apps configuration.
type Config struct {
	// SubscriptionID is the Azure subscription ID.
	SubscriptionID string

	// ResourceGroup is the Azure resource group.
	ResourceGroup string

	// TenantID is the Azure tenant ID.
	TenantID string

	// ClientID is the Azure client/application ID.
	ClientID string

	// ClientSecret is the Azure client secret.
	ClientSecret string

	// Location is the Azure region.
	Location string

	// HTTPClient optionally overrides the HTTP client (tests / custom transport).
	HTTPClient *http.Client

	// SkipAuth skips Azure AD token fetch (for httptest with injected HTTPClient).
	SkipAuth bool

	// ManagementBase overrides the ARM base URL (default https://management.azure.com).
	ManagementBase string

	// LoginBase overrides the AAD login base (default https://login.microsoftonline.com).
	LoginBase string
}

// Engine implements workflow.WorkflowEngine for Azure Logic Apps.
type Engine struct {
	config     Config
	httpClient *http.Client
	token      string
	mu         sync.RWMutex
	workflows  map[string]*workflow.WorkflowDefinition
	executions map[string]*workflow.Execution
	closed     bool
}

// New creates a new Logic Apps engine.
func New(cfg Config) (*Engine, error) {
	if cfg.Location == "" {
		cfg.Location = "eastus"
	}
	if cfg.ManagementBase == "" {
		cfg.ManagementBase = "https://management.azure.com"
	}
	if cfg.LoginBase == "" {
		cfg.LoginBase = "https://login.microsoftonline.com"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}

	engine := &Engine{
		config:     cfg,
		httpClient: hc,
		workflows:  make(map[string]*workflow.WorkflowDefinition),
		executions: make(map[string]*workflow.Execution),
	}

	if !cfg.SkipAuth {
		if err := engine.authenticate(); err != nil {
			return nil, err
		}
	}

	return engine, nil
}

// Close releases local caches. Safe to call multiple times.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	e.executions = nil
	e.workflows = nil
	e.token = ""
	return nil
}

func (e *Engine) checkClosed() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.closed {
		return pkgerrors.FailedPrecondition("logicapps engine is closed", nil)
	}
	return nil
}

func (e *Engine) authenticate() error {
	tokenURL := fmt.Sprintf("%s/%s/oauth2/v2.0/token", e.config.LoginBase, e.config.TenantID)

	data := fmt.Sprintf(
		"client_id=%s&client_secret=%s&scope=https://management.azure.com/.default&grant_type=client_credentials",
		e.config.ClientID, e.config.ClientSecret,
	)

	resp, err := e.httpClient.Post(tokenURL, "application/x-www-form-urlencoded", bytes.NewBufferString(data))
	if err != nil {
		return pkgerrors.Internal("failed to authenticate", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return pkgerrors.Internal("authentication failed: "+string(body), nil)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return pkgerrors.Internal("failed to parse token", err)
	}

	e.token = tokenResp.AccessToken
	return nil
}

func (e *Engine) apiURL(path string) string {
	return fmt.Sprintf(
		"%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Logic%s?api-version=2019-05-01",
		e.config.ManagementBase, e.config.SubscriptionID, e.config.ResourceGroup, path,
	)
}

func (e *Engine) doRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}
	req.Header.Set("Content-Type", "application/json")

	return e.httpClient.Do(req)
}

func (e *Engine) RegisterWorkflow(ctx context.Context, def workflow.WorkflowDefinition) error {
	if err := e.checkClosed(); err != nil {
		return err
	}
	_ = ctx
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := def
	e.workflows[def.ID] = &cp
	return nil
}

func (e *Engine) GetWorkflow(ctx context.Context, workflowID string) (*workflow.WorkflowDefinition, error) {
	if err := e.checkClosed(); err != nil {
		return nil, err
	}
	e.mu.RLock()
	if def, ok := e.workflows[workflowID]; ok {
		cp := *def
		e.mu.RUnlock()
		return &cp, nil
	}
	e.mu.RUnlock()

	url := e.apiURL("/workflows/" + workflowID)
	resp, err := e.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, pkgerrors.Internal("failed to get workflow", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pkgerrors.NotFound("workflow not found", nil)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, pkgerrors.Internal("get workflow failed: "+string(body), nil)
	}

	var result struct {
		Name       string `json:"name"`
		Properties struct {
			CreatedTime time.Time `json:"createdTime"`
			State       string    `json:"state"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	return &workflow.WorkflowDefinition{
		ID:        workflowID,
		Name:      result.Name,
		CreatedAt: result.Properties.CreatedTime,
	}, nil
}

func (e *Engine) Start(ctx context.Context, opts workflow.StartOptions) (*workflow.Execution, error) {
	if err := e.checkClosed(); err != nil {
		return nil, err
	}
	triggerURL := e.apiURL(fmt.Sprintf("/workflows/%s/triggers/manual/run", opts.WorkflowID))

	resp, err := e.doRequest(ctx, "POST", triggerURL, opts.Input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to trigger workflow", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, pkgerrors.Internal("trigger failed: "+string(body), nil)
	}

	execID := resp.Header.Get("x-ms-workflow-run-id")
	if execID == "" {
		execID = resp.Header.Get("x-ms-client-tracking-id")
	}
	if execID == "" {
		if loc := resp.Header.Get("Location"); loc != "" {
			execID = extractRunID(loc)
		}
	}
	if execID == "" {
		execID = uuid.NewString()
	}

	exec := &workflow.Execution{
		ID:         execID,
		WorkflowID: opts.WorkflowID,
		Status:     workflow.StatusRunning,
		Input:      opts.Input,
		StartedAt:  time.Now().UTC(),
	}

	e.mu.Lock()
	e.executions[execID] = exec
	e.mu.Unlock()
	return exec, nil
}

func extractRunID(location string) string {
	// .../workflows/{name}/runs/{runId}
	const marker = "/runs/"
	i := strings.LastIndex(location, marker)
	if i < 0 {
		return ""
	}
	rest := location[i+len(marker):]
	if j := strings.IndexAny(rest, "?/"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func (e *Engine) GetExecution(ctx context.Context, executionID string) (*workflow.Execution, error) {
	if err := e.checkClosed(); err != nil {
		return nil, err
	}

	e.mu.RLock()
	local, ok := e.executions[executionID]
	var workflowID string
	if ok {
		workflowID = local.WorkflowID
	}
	e.mu.RUnlock()

	if workflowID != "" {
		if remote, err := e.fetchRun(ctx, workflowID, executionID); err == nil {
			e.mu.Lock()
			e.executions[executionID] = remote
			e.mu.Unlock()
			return remote, nil
		}
	}

	// Try listing known workflows' runs when we only have a run id.
	e.mu.RLock()
	wfs := make([]string, 0, len(e.workflows))
	for id := range e.workflows {
		wfs = append(wfs, id)
	}
	e.mu.RUnlock()
	for _, wf := range wfs {
		if remote, err := e.fetchRun(ctx, wf, executionID); err == nil {
			e.mu.Lock()
			e.executions[executionID] = remote
			e.mu.Unlock()
			return remote, nil
		}
	}

	if ok {
		cp := *local
		return &cp, nil
	}
	return nil, pkgerrors.NotFound("execution not found", nil)
}

func (e *Engine) fetchRun(ctx context.Context, workflowID, runID string) (*workflow.Execution, error) {
	url := e.apiURL(fmt.Sprintf("/workflows/%s/runs/%s", workflowID, runID))
	resp, err := e.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, pkgerrors.NotFound("run not found", nil)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, pkgerrors.Internal("get run failed: "+string(body), nil)
	}
	var run struct {
		Name       string `json:"name"`
		Properties struct {
			StartTime time.Time `json:"startTime"`
			EndTime   time.Time `json:"endTime"`
			Status    string    `json:"status"`
			Error     *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, pkgerrors.Internal("failed to parse run", err)
	}
	exec := &workflow.Execution{
		ID:          run.Name,
		WorkflowID:  workflowID,
		Status:      mapAzureStatus(run.Properties.Status),
		StartedAt:   run.Properties.StartTime,
		CompletedAt: run.Properties.EndTime,
	}
	if run.Properties.Error != nil {
		exec.Error = run.Properties.Error.Message
	}
	return exec, nil
}

func (e *Engine) ListExecutions(ctx context.Context, opts workflow.ListOptions) (*workflow.ListResult, error) {
	if err := e.checkClosed(); err != nil {
		return nil, err
	}
	if opts.WorkflowID == "" {
		e.mu.RLock()
		defer e.mu.RUnlock()
		result := &workflow.ListResult{Executions: make([]*workflow.Execution, 0, len(e.executions))}
		for _, exec := range e.executions {
			cp := *exec
			result.Executions = append(result.Executions, &cp)
		}
		return result, nil
	}

	url := e.apiURL(fmt.Sprintf("/workflows/%s/runs", opts.WorkflowID))
	resp, err := e.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list runs", err)
	}
	defer resp.Body.Close()

	var runsResp struct {
		Value []struct {
			Name       string `json:"name"`
			Properties struct {
				StartTime time.Time `json:"startTime"`
				EndTime   time.Time `json:"endTime"`
				Status    string    `json:"status"`
			} `json:"properties"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&runsResp); err != nil {
		return nil, pkgerrors.Internal("failed to parse runs", err)
	}

	result := &workflow.ListResult{
		Executions: make([]*workflow.Execution, len(runsResp.Value)),
	}
	for i, run := range runsResp.Value {
		result.Executions[i] = &workflow.Execution{
			ID:          run.Name,
			WorkflowID:  opts.WorkflowID,
			Status:      mapAzureStatus(run.Properties.Status),
			StartedAt:   run.Properties.StartTime,
			CompletedAt: run.Properties.EndTime,
		}
	}

	return result, nil
}

func mapAzureStatus(status string) workflow.ExecutionStatus {
	switch status {
	case "Running", "Waiting":
		return workflow.StatusRunning
	case "Succeeded":
		return workflow.StatusCompleted
	case "Failed":
		return workflow.StatusFailed
	case "Cancelled", "Aborted", "Canceled":
		return workflow.StatusCancelled
	case "TimedOut":
		return workflow.StatusTimedOut
	default:
		return workflow.StatusPending
	}
}

func (e *Engine) Cancel(ctx context.Context, executionID string) error {
	if err := e.checkClosed(); err != nil {
		return err
	}
	e.mu.RLock()
	exec, ok := e.executions[executionID]
	workflowID := ""
	if ok {
		workflowID = exec.WorkflowID
	}
	e.mu.RUnlock()

	if workflowID != "" {
		url := e.apiURL(fmt.Sprintf("/workflows/%s/runs/%s/cancel", workflowID, executionID))
		resp, err := e.doRequest(ctx, "POST", url, nil)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
				e.mu.Lock()
				if exec, ok := e.executions[executionID]; ok {
					exec.Status = workflow.StatusCancelled
					exec.CompletedAt = time.Now().UTC()
				}
				e.mu.Unlock()
				return nil
			}
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if exec, ok := e.executions[executionID]; ok {
		exec.Status = workflow.StatusCancelled
		exec.CompletedAt = time.Now().UTC()
		return nil
	}
	return pkgerrors.NotFound("execution not found", nil)
}

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	_ = ctx
	_ = executionID
	_ = signalName
	_ = data
	return pkgerrors.Unimplemented("signals not supported for Logic Apps (use HTTP webhook actions)", nil)
}

func (e *Engine) Wait(ctx context.Context, executionID string) (*workflow.Execution, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			exec, err := e.GetExecution(ctx, executionID)
			if err != nil {
				return nil, err
			}
			if exec.Status != workflow.StatusRunning && exec.Status != workflow.StatusPending {
				return exec, nil
			}
		}
	}
}
