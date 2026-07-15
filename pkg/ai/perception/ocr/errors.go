package ocr

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for OCR operations.
var (
	ErrEmptyDocument = errors.InvalidArgument("document content or URI is required", nil)
	ErrProvider      = errors.Unavailable("ocr provider unavailable", nil)
)
