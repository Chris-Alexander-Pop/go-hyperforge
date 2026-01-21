package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/perception/vision"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// ComputerVision implements vision.ComputerVision using mock data.
type ComputerVision struct{}

// New creates a new in-memory computer vision client.
func New() *ComputerVision {
	return &ComputerVision{}
}

func (c *ComputerVision) AnalyzeImage(ctx context.Context, image vision.Image, features []vision.Feature) (*vision.Analysis, error) {
	if len(image.Content) == 0 && image.URI == "" {
		return nil, errors.InvalidArgument("image content or URI is required", nil)
	}

	// Mock response
	return &vision.Analysis{
		Labels: []vision.Label{
			{Name: "Cat", Confidence: 0.98},
			{Name: "Animal", Confidence: 0.99},
		},
		SafeSearch: &vision.SafeSearch{
			Adult:    "VERY_UNLIKELY",
			Violence: "VERY_UNLIKELY",
			Racy:     "VERY_UNLIKELY",
		},
	}, nil
}

func (c *ComputerVision) DetectFaces(ctx context.Context, image vision.Image) ([]vision.Face, error) {
	if len(image.Content) == 0 && image.URI == "" {
		return nil, errors.InvalidArgument("image content or URI is required", nil)
	}

	// Mock response
	return []vision.Face{
		{
			BoundingBox: []float64{0.1, 0.1, 0.2, 0.2},
			Confidence:  0.95,
			Landmarks: []vision.Landmark{
				{Type: "LEFT_EYE", X: 0.15, Y: 0.15},
				{Type: "RIGHT_EYE", X: 0.25, Y: 0.15},
			},
		},
	}, nil
}
