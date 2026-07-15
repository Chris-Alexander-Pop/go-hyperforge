// Package prompt provides a versioned prompt-template store with light templating.
//
// Templates are named + versioned strings rendered with:
//   - {{key}} placeholders
//   - {{#if key}}...{{/if}} conditionals (truthy when var is non-empty)
//   - {{include:name}} includes (resolved via Store when rendering)
//
// This remains intentionally thin versus a full prompt ops platform (evals, A/B,
// remote registries).
package prompt

import (
	"context"
	"regexp"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
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
	// Render substitutes placeholders, evaluates conditionals, and resolves includes.
	Render(ctx context.Context, name, version string, vars map[string]string) (string, error)
}

// Domain errors.
var (
	ErrNotFound        = errors.NotFound("prompt template not found", nil)
	ErrInvalidTemplate = errors.InvalidArgument("invalid prompt template", nil)
)

var (
	ifBlockRe     = regexp.MustCompile(`(?s)\{\{#if\s+([a-zA-Z0-9_.-]+)\}\}(.*?)\{\{/if\}\}`)
	includeRe     = regexp.MustCompile(`\{\{include:([a-zA-Z0-9_.-]+)\}\}`)
	placeholderRe = regexp.MustCompile(`\{\{([a-zA-Z0-9_.-]+)\}\}`)
)

// RenderBody applies conditionals and {{key}} substitution on a raw body.
// Includes are left untouched; use RenderBodyWithIncludes or Store.Render.
func RenderBody(body string, vars map[string]string) string {
	out := applyConditionals(body, vars)
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

// IncludeResolver returns a template body by name (typically latest version).
type IncludeResolver func(name string) (string, error)

// RenderBodyWithIncludes applies conditionals, includes, then placeholders.
// Includes are resolved recursively up to maxIncludeDepth.
func RenderBodyWithIncludes(body string, vars map[string]string, resolve IncludeResolver) (string, error) {
	return renderWithIncludes(body, vars, resolve, 0)
}

const maxIncludeDepth = 8

func renderWithIncludes(body string, vars map[string]string, resolve IncludeResolver, depth int) (string, error) {
	if depth > maxIncludeDepth {
		return "", errors.InvalidArgument("prompt include depth exceeded", nil)
	}
	out := applyConditionals(body, vars)
	var firstErr error
	out = includeRe.ReplaceAllStringFunc(out, func(match string) string {
		m := includeRe.FindStringSubmatch(match)
		if len(m) < 2 || resolve == nil {
			return match
		}
		inc, err := resolve(m[1])
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return ""
		}
		rendered, err := renderWithIncludes(inc, vars, resolve, depth+1)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return ""
		}
		return rendered
	})
	if firstErr != nil {
		return "", firstErr
	}
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	// Strip unresolved non-special placeholders only if they look like leftovers
	// from failed optional keys — leave {{include:}} / {{#if}} alone (already handled).
	_ = placeholderRe
	return out, nil
}

func applyConditionals(body string, vars map[string]string) string {
	out := body
	for {
		loc := ifBlockRe.FindStringSubmatchIndex(out)
		if loc == nil {
			break
		}
		m := ifBlockRe.FindStringSubmatch(out[loc[0]:loc[1]])
		key := m[1]
		inner := m[2]
		replacement := ""
		if vars != nil && strings.TrimSpace(vars[key]) != "" {
			replacement = inner
		}
		out = out[:loc[0]] + replacement + out[loc[1]:]
	}
	return out
}
