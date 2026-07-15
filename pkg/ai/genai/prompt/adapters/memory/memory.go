// Package memory is an in-memory versioned prompt template store.
package memory

import (
	"context"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

const latestVersion = "latest"

type versionSet struct {
	latest string
	byVer  map[string]prompt.Template
}

// Store is an in-memory prompt.Store.
type Store struct {
	mu   *concurrency.SmartRWMutex
	data map[string]*versionSet
}

// New creates an empty prompt store.
func New() *Store {
	return &Store{
		mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "prompt-memory"}),
		data: make(map[string]*versionSet),
	}
}

// Put implements prompt.Store.
func (s *Store) Put(ctx context.Context, t prompt.Template) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	t.Name = strings.TrimSpace(t.Name)
	t.Version = strings.TrimSpace(t.Version)
	if t.Name == "" || t.Version == "" || t.Version == latestVersion {
		return prompt.ErrInvalidTemplate
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	vs, ok := s.data[t.Name]
	if !ok {
		vs = &versionSet{byVer: make(map[string]prompt.Template)}
		s.data[t.Name] = vs
	}
	vs.byVer[t.Version] = t
	vs.latest = t.Version
	return nil
}

// Get implements prompt.Store.
func (s *Store) Get(ctx context.Context, name, version string) (*prompt.Template, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	vs, ok := s.data[name]
	if !ok {
		return nil, prompt.ErrNotFound
	}
	ver := version
	if ver == "" || ver == latestVersion {
		ver = vs.latest
	}
	t, ok := vs.byVer[ver]
	if !ok {
		return nil, prompt.ErrNotFound
	}
	cp := t
	return &cp, nil
}

// Render implements prompt.Store.
func (s *Store) Render(ctx context.Context, name, version string, vars map[string]string) (string, error) {
	t, err := s.Get(ctx, name, version)
	if err != nil {
		return "", err
	}
	return prompt.RenderBodyWithIncludes(t.Body, vars, func(incName string) (string, error) {
		inc, err := s.Get(ctx, incName, latestVersion)
		if err != nil {
			return "", err
		}
		return inc.Body, nil
	})
}

var _ prompt.Store = (*Store)(nil)
