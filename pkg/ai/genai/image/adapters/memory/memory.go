// Package memory provides an in-memory image.Service for tests.
package memory

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/image"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Service implements image.Service with mock URLs.
type Service struct {
	baseURL string
}

// New creates a memory image generator. baseURL defaults to "memory://image".
func New(baseURL string) *Service {
	if baseURL == "" {
		baseURL = "memory://image"
	}
	return &Service{baseURL: baseURL}
}

// Generate returns N deterministic mock image URLs for the prompt.
func (s *Service) Generate(ctx context.Context, prompt string, opts image.Options) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if prompt == "" {
		return nil, errors.InvalidArgument("prompt is required", nil)
	}
	n := opts.N
	if n <= 0 {
		n = 1
	}
	urls := make([]string, n)
	for i := 0; i < n; i++ {
		urls[i] = fmt.Sprintf("%s/%d?q=%s", s.baseURL, i, prompt)
	}
	return urls, nil
}

var _ image.Service = (*Service)(nil)
