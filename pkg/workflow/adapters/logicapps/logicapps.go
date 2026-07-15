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
package logicapps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
}

// Engine implements workflow.WorkflowEngine for Azure Logic Apps.
type Engine struct {
	config     Config
	httpClient *http.Client
	token      string
	workflows  map[string]*workflow.WorkflowDefinition
	executions map[string]*workflow.Execution
}

// New creates a new Logic Apps engine.
func New(cfg Config) (*Engine, error) {
	if cfg.Location == "" {
		cfg.Location = "eastus"
	}

	engine := &Engine{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		workflows:  make(map[string]*workflow.WorkflowDefinition),
		executions: make(map[string]*workflow.Execution),
	}

	// Authenticate and get token
	if err := engine.authenticate(); err != nil {
		return nil, err
	}

	return engine, nil
}

func (e *Engine) authenticate() error {
	// OAuth2 token request to Azure AD
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", e.config.TenantID)

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
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Logic%s?api-version=2019-05-01",
		e.config.SubscriptionID, e.config.ResourceGroup, path,
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

	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")

	return e.httpClient.Do(req)
}

func (e *Engine) RegisterWorkflow(ctx context.Context, def workflow.WorkflowDefinition) error {
	// Store workflow definition locally
	// Actual Logic App creation requires ARM template deployment
	e.workflows[def.ID] = &def
	return nil
}

func (e *Engine) GetWorkflow(ctx context.Context, workflowID string) (*workflow.WorkflowDefinition, error) {
	if def, ok := e.workflows[workflowID]; ok {
		return def, nil
	}

	// Try to get from Azure
	url := e.apiURL("/workflows/" + workflowID)
	resp, err := e.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, pkgerrors.Internal("failed to get workflow", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pkgerrors.NotFound("workflow not found", nil)
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
	// Trigger the Logic App via HTTP trigger
	triggerURL := e.apiURL(fmt.Sprintf("/workflows/%s/triggers/manual/run", opts.WorkflowID))

	resp, err := e.doRequest(ctx, "POST", triggerURL, opts.Input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to trigger workflow", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, pkgerrors.Internal("trigger failed: "+string(body), nil)
	}

	execID := uuid.NewString()
	exec := &workflow.Execution{
		ID:         execID,
		WorkflowID: opts.WorkflowID,
		Status:     workflow.StatusRunning,
		Input:      opts.Input,
		StartedAt:  time.Now(),
	}

	e.executions[execID] = exec
	return exec, nil
}

func (e *Engine) GetExecution(ctx context.Context, executionID string) (*workflow.Execution, error) {
	if exec, ok := e.executions[executionID]; ok {
		return exec, nil
	}
	return nil, pkgerrors.NotFound("execution not found", nil)
}

func (e *Engine) ListExecutions(ctx context.Context, opts workflow.ListOptions) (*workflow.ListResult, error) {
	if opts.WorkflowID == "" {
		// List all executions
		result := &workflow.ListResult{Executions: make([]*workflow.Execution, 0)}
		for _, exec := range e.executions {
			result.Executions = append(result.Executions, exec)
		}
		return result, nil
	}

	// Get run history from Azure
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
	case "Cancelled", "Aborted":
		return workflow.StatusCancelled
	case "TimedOut":
		return workflow.StatusTimedOut
	default:
		return workflow.StatusPending
	}
}

func (e *Engine) Cancel(ctx context.Context, executionID string) error {
	if exec, ok := e.executions[executionID]; ok {
		exec.Status = workflow.StatusCancelled
		exec.CompletedAt = time.Now()
		return nil
	}
	return pkgerrors.NotFound("execution not found", nil)
}

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	// Logic Apps doesn't natively support callbacks like Temporal
	return pkgerrors.Internal("signals not supported for Logic Apps", nil)
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
