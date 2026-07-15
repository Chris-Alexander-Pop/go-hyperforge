// Package file provides a JSON-file-backed saga.StateStore.
package file

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga"
)

// Ensure compile-time compliance.
var _ saga.StateStore = (*Store)(nil)

// Store persists each saga execution as a JSON file under Dir.
type Store struct {
	dir string
	mu  *concurrency.SmartMutex
}

// New creates a file/json StateStore rooted at dir (created if missing).
func New(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.InvalidArgument("saga file store dir is required", nil)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Internal("failed to create saga store dir", err)
	}
	return &Store{
		dir: dir,
		mu:  concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "saga-file-store"}),
	}, nil
}

func (s *Store) path(id string) string {
	// Sanitize id for filesystem use.
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, id)
	return filepath.Join(s.dir, safe+".json")
}

// Save writes state as JSON.
func (s *Store) Save(ctx context.Context, state *saga.PersistedState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state == nil || state.ID == "" {
		return errors.InvalidArgument("saga state id is required", nil)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Internal("failed to marshal saga state", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tmp := s.path(state.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return errors.Internal("failed to write saga state", err)
	}
	if err := os.Rename(tmp, s.path(state.ID)); err != nil {
		_ = os.Remove(tmp)
		return errors.Internal("failed to commit saga state", err)
	}
	return nil
}

// Load reads state from JSON.
func (s *Store) Load(ctx context.Context, executionID string) (*saga.PersistedState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path(executionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NotFound("saga execution not found", err)
		}
		return nil, errors.Internal("failed to read saga state", err)
	}
	var state saga.PersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, errors.Internal("failed to unmarshal saga state", err)
	}
	return &state, nil
}

// Delete removes the state file.
func (s *Store) Delete(ctx context.Context, executionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	err := os.Remove(s.path(executionID))
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NotFound("saga execution not found", err)
		}
		return errors.Internal("failed to delete saga state", err)
	}
	return nil
}

// ListIncomplete scans the directory for non-terminal executions.
func (s *Store) ListIncomplete(ctx context.Context) ([]*saga.PersistedState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, errors.Internal("failed to list saga states", err)
	}
	out := make([]*saga.PersistedState, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var state saga.PersistedState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}
		if !state.IsTerminal() {
			cp := state
			out = append(out, &cp)
		}
	}
	return out, nil
}
