package ocr

import (
	"context"
)

// Config configures the OCR service.
type Config struct {
	// Provider specifies the OCR provider (memory, aws-textract, google-vision).
	Provider string `env:"AI_PERCEPTION_OCR_PROVIDER" env-default:"memory"`
}

// Document represents an input document.
type Document struct {
	Content []byte `json:"-"`
	URI     string `json:"uri"`
}

// TextResult represents extracted text.
type TextResult struct {
	Text  string `json:"text"`
	Pages []Page `json:"pages"`
}

// Page represents a page in the document.
type Page struct {
	Number int     `json:"number"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Blocks []Block `json:"blocks"`
}

// Block represents a text block (line, word, paragraph).
type Block struct {
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	Type        string    `json:"type"` // WORD, LINE, PARAGRAPH
	Confidence  float64   `json:"confidence"`
	BoundingBox []float64 `json:"bounding_box"`
}

// OCRClient defines the interface for text extraction.
type OCRClient interface {
	DetectText(ctx context.Context, document Document) (*TextResult, error)
}
