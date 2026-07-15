package memory

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Engine is an in-memory implementation of the template.Engine interface,
// useful for testing. Templates are rendered with text/template.
type Engine struct {
	templates *template.Template
	mu        *concurrency.SmartRWMutex
}

// New creates a new memory template engine.
func New() *Engine {
	return &Engine{
		templates: template.New(""),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-template-engine",
		}),
	}
}

// AddTemplate parses and adds a template to the in-memory store.
func (e *Engine) AddTemplate(name, content string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, err := e.templates.New(name).Parse(content); err != nil {
		return errors.InvalidArgument(fmt.Sprintf("failed to parse template %q", name), err)
	}
	return nil
}

// Render renders a named template with text/template.
func (e *Engine) Render(ctx context.Context, templateName string, data interface{}) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	t := e.templates.Lookup(templateName)
	if t == nil {
		return "", errors.NotFound(fmt.Sprintf("template %s not found", templateName), communication.ErrTemplateNotFound)
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", errors.Internal("failed to render template", err)
	}
	return buf.String(), nil
}

// Close implements the template.Engine interface.
func (e *Engine) Close() error {
	return nil
}
