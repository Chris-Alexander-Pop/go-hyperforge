package template

import (
	"context"
)

// Engine defines the interface for rendering templates.
type Engine interface {
	// Render renders a template with the given data.
	Render(ctx context.Context, templateName string, data interface{}) (string, error)

	// Close releases any resources held by the engine.
	Close() error
}

// Config holds configuration for the Template Engine.
type Config struct {
	// Driver specifies the template backend: "memory", "html", "text".
	Driver string `env:"TEMPLATE_DRIVER" env-default:"memory" validate:"required"`

	// Dir is the directory where templates are stored.
	Dir string `env:"TEMPLATE_DIR"`
}
