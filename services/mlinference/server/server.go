// Package server implements the mlinference service HTTP API.
package server

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/ml/inference"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the mlinference service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"mlinference"`
	Port        string `env:"PORT" env-default:"8115"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the mlinference HTTP API.
type Server struct {
	rest   *rest.Server
	engine inference.InferenceServer
	cfg    Config
}

// New constructs the mlinference HTTP server with an in-memory inference engine.
func New(cfg Config) *Server {
	return NewWithEngine(cfg, inference.NewMemoryServer())
}

// NewWithEngine constructs the server with a custom InferenceServer (tests).
func NewWithEngine(cfg Config, engine inference.InferenceServer) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, engine: engine, cfg: cfg}
	s.routes()
	return s
}

// Echo exposes the underlying Echo instance (tests / custom mounts).
func (s *Server) Echo() *echo.Echo { return s.rest.Echo() }

// Start begins serving HTTP.
func (s *Server) Start() error { return s.rest.Start() }

// Shutdown stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error { return s.rest.Shutdown(ctx) }

func (s *Server) routes() {
	e := s.rest.Echo()
	e.GET("/healthz", s.health)
	e.POST("/v1/inferences", s.predict)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type predictRequest struct {
	ModelID string                 `json:"model_id"`
	Input   map[string]interface{} `json:"input"`
}

type predictResponse struct {
	Output map[string]interface{} `json:"output"`
}

func (s *Server) predict(c echo.Context) error {
	var req predictRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	if req.ModelID == "" {
		return errors.InvalidArgument("model_id is required", nil)
	}
	if req.Input == nil {
		req.Input = map[string]interface{}{}
	}

	ctx := c.Request().Context()
	model, err := s.engine.GetModel(ctx, req.ModelID)
	if err != nil {
		return err
	}
	if model == nil {
		if _, err := s.engine.LoadModel(ctx, inference.Config{
			Name:      req.ModelID,
			ModelPath: "memory://" + req.ModelID,
			ModelType: inference.ModelTypeONNX,
			Version:   "1",
		}); err != nil {
			return err
		}
	}

	resp, err := s.engine.Predict(ctx, &inference.PredictRequest{
		ModelName:  req.ModelID,
		Parameters: req.Input,
	})
	if err != nil {
		return err
	}

	out := map[string]interface{}{
		"model_id":      resp.ModelName,
		"model_version": resp.ModelVersion,
		"echo":          req.Input,
	}
	if len(resp.Outputs) > 0 {
		tensors := make(map[string]interface{}, len(resp.Outputs))
		for name, t := range resp.Outputs {
			tensors[name] = map[string]interface{}{
				"name":      t.Name,
				"shape":     t.Shape,
				"data_type": t.DataType,
			}
		}
		out["tensors"] = tensors
	}
	return c.JSON(http.StatusOK, predictResponse{Output: out})
}
