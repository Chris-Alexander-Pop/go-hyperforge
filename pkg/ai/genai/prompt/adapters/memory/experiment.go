package memory

import (
	"context"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Ensure compile-time compliance.
var (
	_ prompt.Experiment     = (*ABExperiment)(nil)
	_ prompt.RemoteRegistry = (*RemoteRegistry)(nil)
)

// ABExperiment is an in-memory weighted A/B assigner.
type ABExperiment struct {
	mu          *concurrency.SmartRWMutex
	experiments map[string][]prompt.Variant
	assignments map[string]string // experimentID|subjectID → variantID
	outcomes    map[string][]float64
}

// NewABExperiment creates an empty A/B experiment store.
func NewABExperiment() *ABExperiment {
	return &ABExperiment{
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "prompt-ab"}),
		experiments: make(map[string][]prompt.Variant),
		assignments: make(map[string]string),
		outcomes:    make(map[string][]float64),
	}
}

// RegisterVariants registers weighted variants for an experiment.
func (e *ABExperiment) RegisterVariants(experimentID string, variants []prompt.Variant) error {
	if experimentID == "" || len(variants) == 0 {
		return prompt.ErrInvalidTemplate
	}
	cp := make([]prompt.Variant, len(variants))
	copy(cp, variants)
	for i := range cp {
		cp[i].ExperimentID = experimentID
		if cp[i].Weight <= 0 {
			cp[i].Weight = 1
		}
	}
	e.mu.Lock()
	e.experiments[experimentID] = cp
	e.mu.Unlock()
	return nil
}

// Assign implements prompt.Experiment with sticky subject hashing.
func (e *ABExperiment) Assign(ctx context.Context, experimentID, subjectID string) (*prompt.Variant, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := experimentID + "|" + subjectID
	e.mu.Lock()
	defer e.mu.Unlock()
	if vid, ok := e.assignments[key]; ok {
		for _, v := range e.experiments[experimentID] {
			if v.ID == vid {
				cp := v
				return &cp, nil
			}
		}
	}
	vars := e.experiments[experimentID]
	if len(vars) == 0 {
		return nil, errors.NotFound("prompt experiment not found", nil)
	}
	total := 0
	for _, v := range vars {
		total += v.Weight
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	bucket := int(h.Sum32() % uint32(total))
	acc := 0
	chosen := vars[0]
	for _, v := range vars {
		acc += v.Weight
		if bucket < acc {
			chosen = v
			break
		}
	}
	e.assignments[key] = chosen.ID
	cp := chosen
	return &cp, nil
}

// RecordOutcome implements prompt.Experiment.
func (e *ABExperiment) RecordOutcome(ctx context.Context, experimentID, subjectID string, metric float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := experimentID + "|" + subjectID
	e.mu.Lock()
	defer e.mu.Unlock()
	e.outcomes[key] = append(e.outcomes[key], metric)
	return nil
}

// RemoteRegistry is an in-memory remote prompt catalog (pull-through stub).
type RemoteRegistry struct {
	mu   *concurrency.SmartRWMutex
	data map[string]prompt.Template // name@version
}

// NewRemoteRegistry creates an empty remote registry.
func NewRemoteRegistry() *RemoteRegistry {
	return &RemoteRegistry{
		mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "prompt-remote"}),
		data: make(map[string]prompt.Template),
	}
}

// Put registers a remote template (test / seed helper).
func (r *RemoteRegistry) Put(t prompt.Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.Name+"@"+t.Version] = t
}

// Fetch implements prompt.RemoteRegistry.
func (r *RemoteRegistry) Fetch(ctx context.Context, name, version string) (*prompt.Template, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if version == "" || version == latestVersion {
		var latest prompt.Template
		found := false
		for k, t := range r.data {
			if strings.HasPrefix(k, name+"@") {
				if !found || t.Version > latest.Version {
					latest = t
					found = true
				}
			}
		}
		if !found {
			return nil, prompt.ErrNotFound
		}
		cp := latest
		return &cp, nil
	}
	t, ok := r.data[name+"@"+version]
	if !ok {
		return nil, prompt.ErrNotFound
	}
	cp := t
	return &cp, nil
}

// List implements prompt.RemoteRegistry.
func (r *RemoteRegistry) List(ctx context.Context, prefix string) ([]prompt.Template, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]prompt.Template, 0)
	for _, t := range r.data {
		if prefix == "" || strings.HasPrefix(t.Name, prefix) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Version < out[j].Version
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// SyncToStore fetches all remote templates into a local Store.
func (r *RemoteRegistry) SyncToStore(ctx context.Context, store prompt.Store) error {
	list, err := r.List(ctx, "")
	if err != nil {
		return err
	}
	for _, t := range list {
		if err := store.Put(ctx, t); err != nil {
			return err
		}
	}
	return nil
}
