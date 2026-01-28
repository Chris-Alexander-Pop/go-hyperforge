// Package azureml provides an Azure ML training adapter.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/azureml"
//
//	trainer, err := azureml.New(azureml.Config{SubscriptionID: "...", ResourceGroup: "rg", Workspace: "ws"})
//	job, err := trainer.StartJob(ctx, training.JobConfig{...})
package azureml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/training"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds Azure ML configuration.
type Config struct {
	SubscriptionID string
	ResourceGroup  string
	Workspace      string
	TenantID       string
	ClientID       string
	ClientSecret   string
}

// Trainer implements training.Trainer for Azure ML.
type Trainer struct {
	config      Config
	httpClient  *http.Client
	accessToken string
	tokenExpiry time.Time
}

// New creates a new Azure ML trainer.
func New(cfg Config) (*Trainer, error) {
	return &Trainer{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *Trainer) baseURL() string {
	return fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.MachineLearningServices/workspaces/%s",
		t.config.SubscriptionID,
		t.config.ResourceGroup,
		t.config.Workspace,
	)
}

func (t *Trainer) ensureToken(ctx context.Context) error {
	if t.accessToken != "" && time.Now().Before(t.tokenExpiry) {
		return nil
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", t.config.TenantID)

	data := fmt.Sprintf(
		"client_id=%s&client_secret=%s&scope=https://management.azure.com/.default&grant_type=client_credentials",
		t.config.ClientID,
		t.config.ClientSecret,
	)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return pkgerrors.Internal("failed to get access token", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return pkgerrors.Internal("failed to parse token response", err)
	}

	t.accessToken = tokenResp.AccessToken
	t.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return nil
}

func (t *Trainer) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if err := t.ensureToken(ctx); err != nil {
		return nil, err
	}

	url := t.baseURL() + path + "?api-version=2023-04-01"

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+t.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, pkgerrors.Internal(fmt.Sprintf("Azure ML error: %s", string(respBody)), nil)
	}

	return respBody, nil
}

func (t *Trainer) StartJob(ctx context.Context, cfg training.JobConfig) (*training.Job, error) {
	name := cfg.Name
	if name == "" {
		name = fmt.Sprintf("training-%d", time.Now().Unix())
	}

	computeTarget := "cpu-cluster"
	if cfg.InstanceType != "" {
		computeTarget = cfg.InstanceType
	}

	jobDef := map[string]interface{}{
		"properties": map[string]interface{}{
			"jobType":     "Command",
			"displayName": name,
			"command":     fmt.Sprintf("python %s", cfg.EntryPoint),
			"environment": map[string]interface{}{
				"image": cfg.Model,
			},
			"compute": map[string]interface{}{
				"target":        fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.MachineLearningServices/workspaces/%s/computes/%s", t.config.SubscriptionID, t.config.ResourceGroup, t.config.Workspace, computeTarget),
				"instanceCount": cfg.InstanceCount,
			},
			"outputs": map[string]interface{}{
				"default": map[string]interface{}{
					"mode": "rw_mount",
					"uri":  cfg.OutputPath,
				},
			},
		},
	}

	respBody, err := t.doRequest(ctx, "PUT", "/jobs/"+name, jobDef)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Name       string `json:"name"`
		Properties struct {
			Status string `json:"status"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, pkgerrors.Internal("failed to parse job response", err)
	}

	return &training.Job{
		ID:         name,
		Name:       name,
		Status:     training.StatusPending,
		CreatedAt:  time.Now(),
		OutputPath: cfg.OutputPath,
		Config:     cfg,
		Metrics:    make(map[string]float64),
	}, nil
}

func (t *Trainer) GetJob(ctx context.Context, jobID string) (*training.Job, error) {
	respBody, err := t.doRequest(ctx, "GET", "/jobs/"+jobID, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Name       string `json:"name"`
		Properties struct {
			DisplayName string `json:"displayName"`
			Status      string `json:"status"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &training.Job{
		ID:     resp.Name,
		Name:   resp.Properties.DisplayName,
		Status: mapAzureStatus(resp.Properties.Status),
	}, nil
}

func mapAzureStatus(s string) training.JobStatus {
	switch s {
	case "Running":
		return training.StatusRunning
	case "Completed":
		return training.StatusCompleted
	case "Failed":
		return training.StatusFailed
	case "Canceled":
		return training.StatusStopped
	default:
		return training.StatusPending
	}
}

func (t *Trainer) ListJobs(ctx context.Context) ([]*training.Job, error) {
	respBody, err := t.doRequest(ctx, "GET", "/jobs", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Value []struct {
			Name       string `json:"name"`
			Properties struct {
				DisplayName string `json:"displayName"`
				Status      string `json:"status"`
			} `json:"properties"`
		} `json:"value"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	jobs := make([]*training.Job, len(resp.Value))
	for i, j := range resp.Value {
		jobs[i] = &training.Job{
			ID:     j.Name,
			Name:   j.Properties.DisplayName,
			Status: mapAzureStatus(j.Properties.Status),
		}
	}

	return jobs, nil
}

func (t *Trainer) StopJob(ctx context.Context, jobID string) error {
	_, err := t.doRequest(ctx, "POST", "/jobs/"+jobID+"/cancel", nil)
	return err
}

func (t *Trainer) GetMetrics(ctx context.Context, jobID string) ([]training.Metrics, error) {
	return []training.Metrics{}, nil
}

func (t *Trainer) GetLogs(ctx context.Context, jobID string, tail int) ([]string, error) {
	return []string{"Use Azure ML Studio for training logs"}, nil
}

func (t *Trainer) ListCheckpoints(ctx context.Context, jobID string) ([]training.Checkpoint, error) {
	return []training.Checkpoint{}, nil
}

var _ training.Trainer = (*Trainer)(nil)
