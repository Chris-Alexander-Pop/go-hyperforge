// Package mlflow provides an MLflow client for model registry and experiment tracking.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/mlflow"
//
//	client := mlflow.New(mlflow.Config{TrackingURI: "http://localhost:5000"})
//	run, err := client.CreateRun(ctx, "my-experiment")
package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds MLflow configuration.
type Config struct {
	// TrackingURI is the MLflow tracking server URL.
	TrackingURI string

	// Username for authentication.
	Username string

	// Password for authentication.
	Password string
}

// Client provides MLflow operations.
type Client struct {
	config     Config
	httpClient *http.Client
}

// New creates a new MLflow client.
func New(cfg Config) *Client {
	if cfg.TrackingURI == "" {
		cfg.TrackingURI = "http://localhost:5000"
	}

	return &Client{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.config.TrackingURI + "/api/2.0/mlflow" + path

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

	req.Header.Set("Content-Type", "application/json")
	if c.config.Username != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("MLflow request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, pkgerrors.Internal(fmt.Sprintf("MLflow error: %s", string(respBody)), nil)
	}

	return respBody, nil
}

// Experiment represents an MLflow experiment.
type Experiment struct {
	ID               string `json:"experiment_id"`
	Name             string `json:"name"`
	ArtifactLocation string `json:"artifact_location"`
	LifecycleStage   string `json:"lifecycle_stage"`
}

// CreateExperiment creates a new experiment.
func (c *Client) CreateExperiment(ctx context.Context, name string) (*Experiment, error) {
	body := map[string]string{"name": name}
	respBody, err := c.doRequest(ctx, "POST", "/experiments/create", body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ExperimentID string `json:"experiment_id"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &Experiment{ID: resp.ExperimentID, Name: name}, nil
}

// GetExperiment retrieves an experiment by name.
func (c *Client) GetExperiment(ctx context.Context, name string) (*Experiment, error) {
	respBody, err := c.doRequest(ctx, "GET", fmt.Sprintf("/experiments/get-by-name?experiment_name=%s", name), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Experiment Experiment `json:"experiment"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &resp.Experiment, nil
}

// Run represents an MLflow run.
type Run struct {
	ID           string             `json:"run_id"`
	ExperimentID string             `json:"experiment_id"`
	Status       string             `json:"status"`
	StartTime    int64              `json:"start_time"`
	EndTime      int64              `json:"end_time"`
	ArtifactURI  string             `json:"artifact_uri"`
	Metrics      map[string]float64 `json:"metrics"`
	Params       map[string]string  `json:"params"`
	Tags         map[string]string  `json:"tags"`
}

// CreateRun starts a new run.
func (c *Client) CreateRun(ctx context.Context, experimentID string) (*Run, error) {
	body := map[string]interface{}{
		"experiment_id": experimentID,
		"start_time":    time.Now().UnixMilli(),
	}
	respBody, err := c.doRequest(ctx, "POST", "/runs/create", body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Run struct {
			Info struct {
				RunID        string `json:"run_id"`
				ExperimentID string `json:"experiment_id"`
				Status       string `json:"status"`
				ArtifactURI  string `json:"artifact_uri"`
			} `json:"info"`
		} `json:"run"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &Run{
		ID:           resp.Run.Info.RunID,
		ExperimentID: resp.Run.Info.ExperimentID,
		Status:       resp.Run.Info.Status,
		ArtifactURI:  resp.Run.Info.ArtifactURI,
		Metrics:      make(map[string]float64),
		Params:       make(map[string]string),
		Tags:         make(map[string]string),
	}, nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*Run, error) {
	respBody, err := c.doRequest(ctx, "GET", fmt.Sprintf("/runs/get?run_id=%s", runID), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Run struct {
			Info struct {
				RunID        string `json:"run_id"`
				ExperimentID string `json:"experiment_id"`
				Status       string `json:"status"`
				StartTime    int64  `json:"start_time"`
				EndTime      int64  `json:"end_time"`
				ArtifactURI  string `json:"artifact_uri"`
			} `json:"info"`
			Data struct {
				Metrics []struct {
					Key   string  `json:"key"`
					Value float64 `json:"value"`
				} `json:"metrics"`
				Params []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"params"`
			} `json:"data"`
		} `json:"run"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	run := &Run{
		ID:           resp.Run.Info.RunID,
		ExperimentID: resp.Run.Info.ExperimentID,
		Status:       resp.Run.Info.Status,
		StartTime:    resp.Run.Info.StartTime,
		EndTime:      resp.Run.Info.EndTime,
		ArtifactURI:  resp.Run.Info.ArtifactURI,
		Metrics:      make(map[string]float64),
		Params:       make(map[string]string),
	}

	for _, m := range resp.Run.Data.Metrics {
		run.Metrics[m.Key] = m.Value
	}
	for _, p := range resp.Run.Data.Params {
		run.Params[p.Key] = p.Value
	}

	return run, nil
}

// LogMetric logs a metric value.
func (c *Client) LogMetric(ctx context.Context, runID, key string, value float64, step int64) error {
	body := map[string]interface{}{
		"run_id":    runID,
		"key":       key,
		"value":     value,
		"timestamp": time.Now().UnixMilli(),
		"step":      step,
	}
	_, err := c.doRequest(ctx, "POST", "/runs/log-metric", body)
	return err
}

// LogParam logs a parameter.
func (c *Client) LogParam(ctx context.Context, runID, key, value string) error {
	body := map[string]string{
		"run_id": runID,
		"key":    key,
		"value":  value,
	}
	_, err := c.doRequest(ctx, "POST", "/runs/log-parameter", body)
	return err
}

// SetTag sets a tag on a run.
func (c *Client) SetTag(ctx context.Context, runID, key, value string) error {
	body := map[string]string{
		"run_id": runID,
		"key":    key,
		"value":  value,
	}
	_, err := c.doRequest(ctx, "POST", "/runs/set-tag", body)
	return err
}

// EndRun finishes a run.
func (c *Client) EndRun(ctx context.Context, runID, status string) error {
	body := map[string]interface{}{
		"run_id":   runID,
		"status":   status,
		"end_time": time.Now().UnixMilli(),
	}
	_, err := c.doRequest(ctx, "POST", "/runs/update", body)
	return err
}

// RegisteredModel represents a model in the registry.
type RegisteredModel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"creation_timestamp"`
	UpdatedAt   int64  `json:"last_updated_timestamp"`
}

// CreateRegisteredModel creates a model in the registry.
func (c *Client) CreateRegisteredModel(ctx context.Context, name, description string) (*RegisteredModel, error) {
	body := map[string]string{
		"name":        name,
		"description": description,
	}
	respBody, err := c.doRequest(ctx, "POST", "/registered-models/create", body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		RegisteredModel RegisteredModel `json:"registered_model"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &resp.RegisteredModel, nil
}

// ModelVersion represents a model version.
type ModelVersion struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Source       string `json:"source"`
	RunID        string `json:"run_id"`
	Status       string `json:"status"`
	CurrentStage string `json:"current_stage"`
}

// CreateModelVersion creates a model version.
func (c *Client) CreateModelVersion(ctx context.Context, name, source, runID string) (*ModelVersion, error) {
	body := map[string]string{
		"name":   name,
		"source": source,
		"run_id": runID,
	}
	respBody, err := c.doRequest(ctx, "POST", "/model-versions/create", body)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ModelVersion ModelVersion `json:"model_version"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, err
	}

	return &resp.ModelVersion, nil
}

// TransitionModelVersionStage transitions a model version to a new stage.
func (c *Client) TransitionModelVersionStage(ctx context.Context, name, version, stage string) error {
	body := map[string]interface{}{
		"name":                      name,
		"version":                   version,
		"stage":                     stage,
		"archive_existing_versions": true,
	}
	_, err := c.doRequest(ctx, "POST", "/model-versions/transition-stage", body)
	return err
}
