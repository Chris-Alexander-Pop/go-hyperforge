package memory

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Engine is an in-memory implementation of the template.Engine interface.
// useful for testing. It stores templates as strings in a map.
type Engine struct {
	templates map[string]string
	mu        *concurrency.SmartRWMutex
}

// New creates a new memory template engine.
func New() *Engine {
	return &Engine{
		templates: make(map[string]string),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-template-engine",
		}),
	}
}

// AddTemplate adds a template to the in-memory store.
func (e *Engine) AddTemplate(name, content string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.templates[name] = content
}

// Render renders a template by performing simple string interpolation (stub).
// In a real implementation, this might use text/template or html/template.
func (e *Engine) Render(ctx context.Context, templateName string, data interface{}) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	content, ok := e.templates[templateName]
	if !ok {
		return "", errors.NotFound(fmt.Sprintf("template %s not found", templateName), nil)
	}

	// For simple testing, we just return the content as is, or maybe format it with data if it's a string.
	// This is a very basic mock.
	return fmt.Sprintf("%s - %v", content, data), nil
}

// Close implements the template.Engine interface.
func (e *Engine) Close() error {
	return nil
}
