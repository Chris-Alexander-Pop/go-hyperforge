// Package prompt provides a thin prompt-template store with version stubs.
//
// Templates are named + versioned strings rendered with simple {{key}} placeholders.
// This is a skeleton for a fuller prompt engine (evals, A/B, remote registries).
package prompt

import (
	"context"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Template is a versioned prompt body.
type Template struct {
	Name    string
	Version string
	Body    string
}

// Store retrieves and renders versioned prompt templates.
type Store interface {
	// Put registers or replaces a template version.
	Put(ctx context.Context, t Template) error
	// Get returns a specific version, or the latest when version is empty/"latest".
	Get(ctx context.Context, name, version string) (*Template, error)
	// Render substitutes {{key}} placeholders from vars.
	Render(ctx context.Context, name, version string, vars map[string]string) (string, error)
}

// Domain errors.
var (
	ErrNotFound        = errors.NotFound("prompt template not found", nil)
	ErrInvalidTemplate = errors.InvalidArgument("invalid prompt template", nil)
)

// RenderBody applies {{key}} substitution on a raw body.
func RenderBody(body string, vars map[string]string) string {
	out := body
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}
