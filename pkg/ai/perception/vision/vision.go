package vision

import (
	"context"
)

// Config configures the computer vision service.
type Config struct {
	// Provider specifies the vision provider (memory, aws-rekognition, google-vision).
	Provider string `env:"AI_PERCEPTION_VISION_PROVIDER" env-default:"memory"`
}

// Feature represents a type of analysis feature to extract.
type Feature string

const (
	FeatureLabels     Feature = "LABELS"
	FeatureFaces      Feature = "FACES"
	FeatureSafeSearch Feature = "SAFE_SEARCH"
	FeatureText       Feature = "TEXT"
)

// Image represents an image input, either as raw bytes or a URI.
type Image struct {
	Content []byte `json:"-"`
	URI     string `json:"uri"`
}

// Analysis represents the result of a general image analysis.
type Analysis struct {
	Labels     []Label     `json:"labels,omitempty"`
	SafeSearch *SafeSearch `json:"safe_search,omitempty"`
}

// Label represents a detected entity or concept.
type Label struct {
	Name       string   `json:"name"`
	Confidence float64  `json:"confidence"`
	Parents    []string `json:"parents,omitempty"`
}

// SafeSearch represents moderation detection results.
type SafeSearch struct {
	Adult    string `json:"adult"`    // UNKNOWN, VERY_UNLIKELY, UNLIKELY, POSSIBLE, LIKELY, VERY_LIKELY
	Violence string `json:"violence"` // UNKNOWN, VERY_UNLIKELY, UNLIKELY, POSSIBLE, LIKELY, VERY_LIKELY
	Racy     string `json:"racy"`     // UNKNOWN, VERY_UNLIKELY, UNLIKELY, POSSIBLE, LIKELY, VERY_LIKELY
}

// Face represents a detected face.
type Face struct {
	BoundingBox []float64  `json:"bounding_box"` // x, y, width, height (normalized 0-1)
	Confidence  float64    `json:"confidence"`
	Landmarks   []Landmark `json:"landmarks,omitempty"`
}

// Landmark represents a facial feature.
type Landmark struct {
	Type string  `json:"type"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// ComputerVision defines the interface for image analysis operations.
type ComputerVision interface {
	AnalyzeImage(ctx context.Context, image Image, features []Feature) (*Analysis, error)
	DetectFaces(ctx context.Context, image Image) ([]Face, error)
}
