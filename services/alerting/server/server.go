// Package server implements the alerting service HTTP API.
package server

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Config is the alerting service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"alerting"`
	Port        string `env:"PORT" env-default:"8105"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Rule is an alert rule definition.
type Rule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	Severity  string    `json:"severity"`
	CreatedAt time.Time `json:"created_at"`
}

// Alert is a fired alert instance.
type Alert struct {
	ID      string    `json:"id"`
	RuleID  string    `json:"rule_id"`
	Message string    `json:"message"`
	Status  string    `json:"status"`
	FiredAt time.Time `json:"fired_at"`
	AckedAt time.Time `json:"acked_at,omitempty"`
}

// Server wraps the alerts HTTP API.
type Server struct {
	rest   *rest.Server
	cfg    Config
	mu     sync.RWMutex
	rules  map[string]Rule
	alerts []Alert
}

// New constructs the alerting HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{rest: r, cfg: cfg, rules: make(map[string]Rule), alerts: make([]Alert, 0)}
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
	e.POST("/v1/alerts/rules", s.createRule)
	e.POST("/v1/alerts/fire", s.fire)
	e.GET("/v1/alerts", s.list)
	e.POST("/v1/alerts/:id/ack", s.ack)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type createRuleRequest struct {
	Name     string `json:"name"`
	Query    string `json:"query"`
	Severity string `json:"severity"`
}

func (s *Server) createRule(c echo.Context) error {
	var req createRuleRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.InvalidArgument("name is required", nil)
	}
	sev := strings.ToLower(strings.TrimSpace(req.Severity))
	if sev == "" {
		sev = "warning"
	}
	rule := Rule{
		ID:        uuid.NewString(),
		Name:      name,
		Query:     strings.TrimSpace(req.Query),
		Severity:  sev,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.rules[rule.ID] = rule
	s.mu.Unlock()
	return c.JSON(http.StatusCreated, rule)
}

type fireRequest struct {
	RuleID  string `json:"rule_id"`
	Message string `json:"message"`
}

func (s *Server) fire(c echo.Context) error {
	var req fireRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	ruleID := strings.TrimSpace(req.RuleID)
	msg := strings.TrimSpace(req.Message)
	if ruleID == "" {
		return errors.InvalidArgument("rule_id is required", nil)
	}
	if msg == "" {
		return errors.InvalidArgument("message is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[ruleID]; !ok {
		return errors.NotFound("rule not found", nil)
	}
	a := Alert{
		ID:      uuid.NewString(),
		RuleID:  ruleID,
		Message: msg,
		Status:  "firing",
		FiredAt: time.Now().UTC(),
	}
	s.alerts = append(s.alerts, a)
	return c.JSON(http.StatusCreated, a)
}

func (s *Server) list(c echo.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Alert, len(s.alerts))
	copy(out, s.alerts)
	return c.JSON(http.StatusOK, map[string]interface{}{"alerts": out})
}

func (s *Server) ack(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.alerts {
		if s.alerts[i].ID == id {
			s.alerts[i].Status = "acked"
			s.alerts[i].AckedAt = time.Now().UTC()
			return c.JSON(http.StatusOK, s.alerts[i])
		}
	}
	return errors.NotFound("alert not found", nil)
}
