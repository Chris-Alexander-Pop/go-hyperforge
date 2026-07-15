// Package memory provides an in-memory implementation of apigateway.APIGatewayManager.
package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/apigateway"
	"github.com/google/uuid"
)

// Manager implements an in-memory API gateway manager for testing.
type Manager struct {
	mu   *concurrency.SmartRWMutex
	apis map[string]*apigateway.API
}

// New creates a new in-memory API gateway manager.
func New() *Manager {
	return &Manager{
		mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-apigateway"}),
		apis: make(map[string]*apigateway.API),
	}
}

func (m *Manager) CreateAPI(ctx context.Context, opts apigateway.CreateAPIOptions) (*apigateway.API, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.NewString()[:10]
	api := &apigateway.API{
		ID:          id,
		Name:        opts.Name,
		Description: opts.Description,
		Type:        opts.Type,
		Endpoint:    fmt.Sprintf("https://%s.execute-api.us-east-1.amazonaws.com", id),
		Version:     opts.Version,
		Routes:      []apigateway.Route{},
		Stages:      []apigateway.Stage{},
		Tags:        opts.Tags,
		CreatedAt:   time.Now(),
	}

	if api.Type == "" {
		api.Type = apigateway.APITypeHTTP
	}

	m.apis[id] = api
	return api, nil
}

func (m *Manager) GetAPI(ctx context.Context, id string) (*apigateway.API, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	api, ok := m.apis[id]
	if !ok {
		return nil, apigateway.ErrAPINotFound
	}
	return api, nil
}

func (m *Manager) ListAPIs(ctx context.Context) ([]*apigateway.API, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*apigateway.API, 0, len(m.apis))
	for _, api := range m.apis {
		result = append(result, api)
	}
	return result, nil
}

func (m *Manager) DeleteAPI(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.apis[id]; !ok {
		return apigateway.ErrAPINotFound
	}
	delete(m.apis, id)
	return nil
}

func (m *Manager) AddRoute(ctx context.Context, apiID string, route apigateway.Route) (*apigateway.Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	api, ok := m.apis[apiID]
	if !ok {
		return nil, apigateway.ErrAPINotFound
	}

	route.ID = uuid.NewString()[:8]
	api.Routes = append(api.Routes, route)
	return &route, nil
}

func (m *Manager) RemoveRoute(ctx context.Context, apiID, routeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	api, ok := m.apis[apiID]
	if !ok {
		return apigateway.ErrAPINotFound
	}

	for i, r := range api.Routes {
		if r.ID == routeID {
			api.Routes = append(api.Routes[:i], api.Routes[i+1:]...)
			return nil
		}
	}
	return apigateway.ErrRouteNotFound
}

func (m *Manager) Deploy(ctx context.Context, apiID, stageName string) (*apigateway.Stage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	api, ok := m.apis[apiID]
	if !ok {
		return nil, apigateway.ErrAPINotFound
	}

	// Check if stage exists, update it
	for i, s := range api.Stages {
		if s.Name == stageName {
			api.Stages[i].DeployedAt = time.Now()
			return &api.Stages[i], nil
		}
	}

	// Create new stage
	stage := apigateway.Stage{
		Name:       stageName,
		Variables:  make(map[string]string),
		DeployedAt: time.Now(),
	}
	api.Stages = append(api.Stages, stage)
	return &stage, nil
}

func (m *Manager) GetStage(ctx context.Context, apiID, stageName string) (*apigateway.Stage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	api, ok := m.apis[apiID]
	if !ok {
		return nil, apigateway.ErrAPINotFound
	}

	for _, s := range api.Stages {
		if s.Name == stageName {
			return &s, nil
		}
	}
	return nil, apigateway.ErrStageNotFound
}
