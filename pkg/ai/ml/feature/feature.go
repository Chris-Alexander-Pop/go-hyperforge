// Package feature provides a feature store client for ML features.
//
// Supports feature retrieval for training and inference.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/feature"
//
//	store := feature.New(feature.Config{})
//	features, err := store.GetOnlineFeatures(ctx, "user-features", entityKeys)
package feature

import (
	"context"
	"sync"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds feature store configuration.
type Config struct {
	// Backend specifies the storage backend.
	Backend string // "memory", "redis", "feast", "sagemaker"

	// Endpoint for remote feature stores.
	Endpoint string

	// Credentials for authentication.
	Credentials map[string]string
}

// FeatureGroup represents a group of related features.
type FeatureGroup struct {
	// Name is the feature group name.
	Name string

	// EntityKey is the primary key column.
	EntityKey string

	// Features in this group.
	Features []FeatureDefinition

	// TTL for online features.
	TTL time.Duration

	// Description of the feature group.
	Description string

	// Tags for organization.
	Tags map[string]string

	// CreatedAt is when the group was created.
	CreatedAt time.Time

	// UpdatedAt is when features were last updated.
	UpdatedAt time.Time
}

// FeatureDefinition describes a feature.
type FeatureDefinition struct {
	// Name is the feature name.
	Name string

	// Type is the feature data type.
	Type FeatureType

	// Description of the feature.
	Description string

	// DefaultValue when feature is missing.
	DefaultValue interface{}
}

// FeatureType represents feature data types.
type FeatureType string

const (
	FeatureTypeFloat  FeatureType = "float"
	FeatureTypeInt    FeatureType = "int"
	FeatureTypeString FeatureType = "string"
	FeatureTypeBool   FeatureType = "bool"
	FeatureTypeList   FeatureType = "list"
	FeatureTypeVector FeatureType = "vector"
)

// FeatureVector is a set of feature values.
type FeatureVector struct {
	// EntityKey is the entity identifier.
	EntityKey string

	// Features is the feature name to value mapping.
	Features map[string]interface{}

	// EventTime is when features were computed.
	EventTime time.Time

	// CreatedAt is when stored.
	CreatedAt time.Time
}

// FeatureStore manages feature storage and retrieval.
type FeatureStore interface {
	// CreateFeatureGroup creates a new feature group.
	CreateFeatureGroup(ctx context.Context, group *FeatureGroup) error

	// GetFeatureGroup retrieves a feature group.
	GetFeatureGroup(ctx context.Context, name string) (*FeatureGroup, error)

	// ListFeatureGroups returns all feature groups.
	ListFeatureGroups(ctx context.Context) ([]*FeatureGroup, error)

	// DeleteFeatureGroup removes a feature group.
	DeleteFeatureGroup(ctx context.Context, name string) error

	// IngestFeatures stores feature values.
	IngestFeatures(ctx context.Context, groupName string, vectors []FeatureVector) error

	// GetOnlineFeatures retrieves features for real-time inference.
	GetOnlineFeatures(ctx context.Context, groupName string, entityKeys []string, featureNames []string) ([]FeatureVector, error)

	// GetHistoricalFeatures retrieves features for training.
	GetHistoricalFeatures(ctx context.Context, groupName string, entityKeys []string, startTime, endTime time.Time) ([]FeatureVector, error)
}

// MemoryStore is an in-memory feature store.
type MemoryStore struct {
	groups   map[string]*FeatureGroup
	features map[string]map[string]*FeatureVector // group -> entityKey -> vector
	mu       sync.RWMutex
}

// New creates a new memory feature store.
func New(cfg Config) *MemoryStore {
	return &MemoryStore{
		groups:   make(map[string]*FeatureGroup),
		features: make(map[string]map[string]*FeatureVector),
	}
}

func (s *MemoryStore) CreateFeatureGroup(ctx context.Context, group *FeatureGroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[group.Name]; exists {
		return pkgerrors.Conflict("feature group already exists", nil)
	}

	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	s.groups[group.Name] = group
	s.features[group.Name] = make(map[string]*FeatureVector)

	return nil
}

func (s *MemoryStore) GetFeatureGroup(ctx context.Context, name string) (*FeatureGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	group, ok := s.groups[name]
	if !ok {
		return nil, pkgerrors.NotFound("feature group not found", nil)
	}
	return group, nil
}

func (s *MemoryStore) ListFeatureGroups(ctx context.Context) ([]*FeatureGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groups := make([]*FeatureGroup, 0, len(s.groups))
	for _, g := range s.groups {
		groups = append(groups, g)
	}
	return groups, nil
}

func (s *MemoryStore) DeleteFeatureGroup(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.groups, name)
	delete(s.features, name)
	return nil
}

func (s *MemoryStore) IngestFeatures(ctx context.Context, groupName string, vectors []FeatureVector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	group, ok := s.groups[groupName]
	if !ok {
		return pkgerrors.NotFound("feature group not found", nil)
	}

	groupFeatures := s.features[groupName]
	for _, v := range vectors {
		v.CreatedAt = time.Now()
		groupFeatures[v.EntityKey] = &v
	}

	group.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) GetOnlineFeatures(ctx context.Context, groupName string, entityKeys []string, featureNames []string) ([]FeatureVector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groupFeatures, ok := s.features[groupName]
	if !ok {
		return nil, pkgerrors.NotFound("feature group not found", nil)
	}

	result := make([]FeatureVector, len(entityKeys))
	for i, key := range entityKeys {
		vec, ok := groupFeatures[key]
		if !ok {
			result[i] = FeatureVector{EntityKey: key, Features: make(map[string]interface{})}
			continue
		}

		// Filter to requested features
		filtered := FeatureVector{
			EntityKey: vec.EntityKey,
			EventTime: vec.EventTime,
			Features:  make(map[string]interface{}),
		}

		if len(featureNames) == 0 {
			filtered.Features = vec.Features
		} else {
			for _, name := range featureNames {
				if val, exists := vec.Features[name]; exists {
					filtered.Features[name] = val
				}
			}
		}
		result[i] = filtered
	}

	return result, nil
}

func (s *MemoryStore) GetHistoricalFeatures(ctx context.Context, groupName string, entityKeys []string, startTime, endTime time.Time) ([]FeatureVector, error) {
	// For memory store, just return online features
	return s.GetOnlineFeatures(ctx, groupName, entityKeys, nil)
}

var _ FeatureStore = (*MemoryStore)(nil)
