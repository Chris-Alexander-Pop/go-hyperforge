package text

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication"
	tmpl "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/template"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Engine renders templates using the standard library text/template package.
type Engine struct {
	templates *template.Template
	mu        *concurrency.SmartRWMutex
}

// New creates a text/template engine. When cfg.Dir is set, all *.tmpl and *.txt
// files in that directory are parsed (basename without extension is the name).
func New(cfg tmpl.Config) (*Engine, error) {
	e := &Engine{
		templates: template.New(""),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "text-template-engine",
		}),
	}

	if cfg.Dir != "" {
		if err := e.loadDir(cfg.Dir); err != nil {
			return nil, err
		}
	}

	return e, nil
}

// NewFromString parses a single named template from content.
func NewFromString(name, content string) (*Engine, error) {
	e := &Engine{
		templates: template.New(""),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "text-template-engine",
		}),
	}
	if err := e.AddTemplate(name, content); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) loadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.InvalidArgument("failed to read template directory", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".tmpl" && ext != ".txt" && ext != ".gotmpl" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return errors.Internal("failed to read template file", err)
		}
		tmplName := strings.TrimSuffix(name, ext)
		if _, err := e.templates.New(tmplName).Parse(string(content)); err != nil {
			return errors.InvalidArgument(fmt.Sprintf("failed to parse template %q", tmplName), err)
		}
	}
	return nil
}

// AddTemplate parses and registers a named template.
func (e *Engine) AddTemplate(name, content string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, err := e.templates.New(name).Parse(content); err != nil {
		return errors.InvalidArgument(fmt.Sprintf("failed to parse template %q", name), err)
	}
	return nil
}

// Render renders a named template with the given data.
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

// Close implements template.Engine.
func (e *Engine) Close() error {
	return nil
}

var _ tmpl.Engine = (*Engine)(nil)
