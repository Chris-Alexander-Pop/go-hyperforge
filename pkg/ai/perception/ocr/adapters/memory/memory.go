package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/perception/ocr"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// OCRClient implements ocr.OCRClient using mock data.
type OCRClient struct{}

// New creates a new in-memory OCR client.
func New() *OCRClient {
	return &OCRClient{}
}

func (c *OCRClient) DetectText(ctx context.Context, document ocr.Document) (*ocr.TextResult, error) {
	if len(document.Content) == 0 && document.URI == "" {
		return nil, errors.InvalidArgument("document content or URI is required", nil)
	}

	// Mock response
	return &ocr.TextResult{
		Text: "This is a mock OCR document result.",
		Pages: []ocr.Page{
			{
				Number: 1,
				Width:  8.5,
				Height: 11.0,
				Blocks: []ocr.Block{
					{
						ID:          "block-1",
						Text:        "This is a mock OCR document result.",
						Type:        "LINE",
						Confidence:  0.99,
						BoundingBox: []float64{0.1, 0.1, 0.8, 0.1},
					},
				},
			},
		},
	}, nil
}
