// Package server implements the recommendation service HTTP API.
package server

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/labstack/echo/v4"
)

// Config is the recommendation service environment configuration.
type Config struct {
	ServiceName string `env:"SERVICE_NAME" env-default:"recommendation"`
	Port        string `env:"PORT" env-default:"8116"`
	LogLevel    string `env:"LOG_LEVEL" env-default:"info"`
}

// Server wraps the recommendations HTTP API with simple popularity + co-occurrence.
type Server struct {
	rest *rest.Server
	cfg  Config
	mu   sync.RWMutex
	// item popularity counts
	popular map[string]int
	// user -> set of items interacted with
	userItems map[string]map[string]struct{}
	// item -> co-occurrence counts with other items
	cooccur map[string]map[string]int
}

// New constructs the recommendation HTTP server.
func New(cfg Config) *Server {
	r := rest.New(rest.Config{Port: cfg.Port})
	s := &Server{
		rest:      r,
		cfg:       cfg,
		popular:   make(map[string]int),
		userItems: make(map[string]map[string]struct{}),
		cooccur:   make(map[string]map[string]int),
	}
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
	e.POST("/v1/recommendations/interactions", s.track)
	e.POST("/v1/recommendations", s.recommend)
}

func (s *Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type trackRequest struct {
	UserID string `json:"user_id"`
	ItemID string `json:"item_id"`
}

func (s *Server) track(c echo.Context) error {
	var req trackRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	userID := strings.TrimSpace(req.UserID)
	itemID := strings.TrimSpace(req.ItemID)
	if userID == "" {
		return errors.InvalidArgument("user_id is required", nil)
	}
	if itemID == "" {
		return errors.InvalidArgument("item_id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.popular[itemID]++
	if _, ok := s.userItems[userID]; !ok {
		s.userItems[userID] = make(map[string]struct{})
	}
	for other := range s.userItems[userID] {
		if other == itemID {
			continue
		}
		if _, ok := s.cooccur[itemID]; !ok {
			s.cooccur[itemID] = make(map[string]int)
		}
		if _, ok := s.cooccur[other]; !ok {
			s.cooccur[other] = make(map[string]int)
		}
		s.cooccur[itemID][other]++
		s.cooccur[other][itemID]++
	}
	s.userItems[userID][itemID] = struct{}{}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"user_id": userID,
		"item_id": itemID,
		"tracked": true,
	})
}

type recommendRequest struct {
	UserID string `json:"user_id"`
	Limit  int    `json:"limit,omitempty"`
}

type scoredItem struct {
	ItemID string `json:"item_id"`
	Score  int    `json:"score"`
}

func (s *Server) recommend(c echo.Context) error {
	var req recommendRequest
	if err := c.Bind(&req); err != nil {
		return errors.InvalidArgument("invalid JSON body", err)
	}
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return errors.InvalidArgument("user_id is required", nil)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := s.userItems[userID]
	scores := make(map[string]int)
	for item := range seen {
		for other, w := range s.cooccur[item] {
			if _, ok := seen[other]; ok {
				continue
			}
			scores[other] += w
		}
	}
	// Fall back to popularity for cold start / fill.
	type kv struct {
		id string
		sc int
	}
	var ranked []kv
	for id, sc := range scores {
		ranked = append(ranked, kv{id, sc})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].sc == ranked[j].sc {
			return ranked[i].id < ranked[j].id
		}
		return ranked[i].sc > ranked[j].sc
	})
	if len(ranked) < limit {
		var pop []kv
		for id, sc := range s.popular {
			if _, ok := seen[id]; ok {
				continue
			}
			if _, ok := scores[id]; ok {
				continue
			}
			pop = append(pop, kv{id, sc})
		}
		sort.Slice(pop, func(i, j int) bool {
			if pop[i].sc == pop[j].sc {
				return pop[i].id < pop[j].id
			}
			return pop[i].sc > pop[j].sc
		})
		ranked = append(ranked, pop...)
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]scoredItem, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, scoredItem{ItemID: r.id, Score: r.sc})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":         userID,
		"recommendations": out,
	})
}
