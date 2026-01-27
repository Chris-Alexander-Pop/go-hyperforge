// Package inference provides model serving for ML inference.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/ml/inference"
//
//	server := inference.New(inference.Config{ModelPath: "/models/resnet50"})
//	result, err := server.Predict(ctx, input)
package inference

import (
	"context"
	"sync"
	"time"
)

// ModelType identifies the model framework.
type ModelType string

const (
	ModelTypeTensorFlow ModelType = "tensorflow"
	ModelTypePyTorch    ModelType = "pytorch"
	ModelTypeONNX       ModelType = "onnx"
	ModelTypeTensorRT   ModelType = "tensorrt"
	ModelTypeTriton     ModelType = "triton"
	ModelTypeSageMaker  ModelType = "sagemaker"
)

// Config holds inference server configuration.
type Config struct {
	// Name is the model name.
	Name string

	// ModelPath is the path to the model.
	ModelPath string

	// ModelType is the framework.
	ModelType ModelType

	// Version is the model version.
	Version string

	// BatchSize for batching requests.
	BatchSize int

	// MaxBatchDelay is max time to wait for batching.
	MaxBatchDelay time.Duration

	// Timeout for inference requests.
	Timeout time.Duration

	// GPU enables GPU inference.
	GPU bool

	// DeviceID is the GPU device.
	DeviceID int
}

// Model represents a loaded model.
type Model struct {
	// Name is the model name.
	Name string

	// Version is the model version.
	Version string

	// Type is the model framework.
	Type ModelType

	// Path is the model location.
	Path string

	// Status is the model state.
	Status ModelStatus

	// LoadedAt is when the model was loaded.
	LoadedAt time.Time

	// Metadata about the model.
	Metadata map[string]interface{}
}

// ModelStatus represents model state.
type ModelStatus string

const (
	ModelStatusLoading   ModelStatus = "loading"
	ModelStatusReady     ModelStatus = "ready"
	ModelStatusUnloading ModelStatus = "unloading"
	ModelStatusError     ModelStatus = "error"
)

// PredictRequest is an inference request.
type PredictRequest struct {
	// ModelName is the target model.
	ModelName string

	// ModelVersion is the specific version (optional).
	ModelVersion string

	// Inputs are the input tensors.
	Inputs map[string]Tensor

	// Parameters for inference.
	Parameters map[string]interface{}
}

// Tensor represents an ML tensor.
type Tensor struct {
	// Name is the tensor name.
	Name string

	// Shape is the tensor dimensions.
	Shape []int64

	// DataType is the element type.
	DataType DataType

	// Data is the raw data.
	Data []byte
}

// DataType represents tensor element types.
type DataType string

const (
	DataTypeFloat32 DataType = "float32"
	DataTypeFloat64 DataType = "float64"
	DataTypeInt32   DataType = "int32"
	DataTypeInt64   DataType = "int64"
	DataTypeUint8   DataType = "uint8"
	DataTypeString  DataType = "string"
	DataTypeBool    DataType = "bool"
)

// PredictResponse is the inference result.
type PredictResponse struct {
	// ModelName is the model used.
	ModelName string

	// ModelVersion is the version used.
	ModelVersion string

	// Outputs are the output tensors.
	Outputs map[string]Tensor

	// InferenceTime is the inference duration.
	InferenceTime time.Duration
}

// InferenceServer serves model predictions.
type InferenceServer interface {
	// LoadModel loads a model for serving.
	LoadModel(ctx context.Context, config Config) (*Model, error)

	// UnloadModel removes a model.
	UnloadModel(ctx context.Context, name string) error

	// GetModel retrieves model info.
	GetModel(ctx context.Context, name string) (*Model, error)

	// ListModels returns all loaded models.
	ListModels(ctx context.Context) ([]*Model, error)

	// Predict runs inference.
	Predict(ctx context.Context, request *PredictRequest) (*PredictResponse, error)

	// PredictBatch runs batched inference.
	PredictBatch(ctx context.Context, requests []*PredictRequest) ([]*PredictResponse, error)

	// Health returns server health status.
	Health(ctx context.Context) (*HealthStatus, error)
}

// HealthStatus represents server health.
type HealthStatus struct {
	Healthy          bool
	Message          string
	ModelsLoaded     int
	RequestsServed   int64
	AverageLatencyMs float64
}

// MemoryServer is an in-memory inference server for testing.
type MemoryServer struct {
	models map[string]*Model
	mu     sync.RWMutex
	stats  ServerStats
}

// ServerStats tracks server statistics.
type ServerStats struct {
	RequestsServed int64
	TotalLatencyNs int64
}

// NewMemoryServer creates an in-memory inference server.
func NewMemoryServer() *MemoryServer {
	return &MemoryServer{
		models: make(map[string]*Model),
	}
}

func (s *MemoryServer) LoadModel(ctx context.Context, config Config) (*Model, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	model := &Model{
		Name:     config.Name,
		Version:  config.Version,
		Type:     config.ModelType,
		Path:     config.ModelPath,
		Status:   ModelStatusReady,
		LoadedAt: time.Now(),
		Metadata: make(map[string]interface{}),
	}

	s.models[config.Name] = model
	return model, nil
}

func (s *MemoryServer) UnloadModel(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.models, name)
	return nil
}

func (s *MemoryServer) GetModel(ctx context.Context, name string) (*Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, ok := s.models[name]
	if !ok {
		return nil, nil
	}
	return model, nil
}

func (s *MemoryServer) ListModels(ctx context.Context) ([]*Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	models := make([]*Model, 0, len(s.models))
	for _, m := range s.models {
		models = append(models, m)
	}
	return models, nil
}

func (s *MemoryServer) Predict(ctx context.Context, req *PredictRequest) (*PredictResponse, error) {
	start := time.Now()
	s.stats.RequestsServed++

	// Simulate inference
	resp := &PredictResponse{
		ModelName:     req.ModelName,
		ModelVersion:  req.ModelVersion,
		Outputs:       make(map[string]Tensor),
		InferenceTime: time.Since(start),
	}

	s.stats.TotalLatencyNs += resp.InferenceTime.Nanoseconds()
	return resp, nil
}

func (s *MemoryServer) PredictBatch(ctx context.Context, requests []*PredictRequest) ([]*PredictResponse, error) {
	responses := make([]*PredictResponse, len(requests))
	for i, req := range requests {
		resp, err := s.Predict(ctx, req)
		if err != nil {
			return nil, err
		}
		responses[i] = resp
	}
	return responses, nil
}

func (s *MemoryServer) Health(ctx context.Context) (*HealthStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	avgLatency := float64(0)
	if s.stats.RequestsServed > 0 {
		avgLatency = float64(s.stats.TotalLatencyNs) / float64(s.stats.RequestsServed) / 1e6
	}

	return &HealthStatus{
		Healthy:          true,
		ModelsLoaded:     len(s.models),
		RequestsServed:   s.stats.RequestsServed,
		AverageLatencyMs: avgLatency,
	}, nil
}

var _ InferenceServer = (*MemoryServer)(nil)
