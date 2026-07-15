// Package gateway provides a multi-provider LLM client router with fallback.
//
// Router implements llm.Client: Chat and StreamChat try providers in order
// until one succeeds. Use this to fan across OpenAI / Anthropic / memory
// without changing call sites.
package gateway

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Provider is a named llm.Client used by the router.
type Provider struct {
	Name   string
	Client llm.Client
}

// Router tries providers in registration order on failure.
type Router struct {
	providers []Provider
}

// New builds a router. At least one non-nil client is required.
// The first argument is primary; additional clients are fallbacks.
func New(primary llm.Client, fallbacks ...llm.Client) (*Router, error) {
	if primary == nil {
		return nil, llm.ErrNilClient
	}
	r := &Router{
		providers: []Provider{{Name: "primary", Client: primary}},
	}
	for i, fb := range fallbacks {
		if fb == nil {
			continue
		}
		r.providers = append(r.providers, Provider{
			Name:   fmt.Sprintf("fallback-%d", i),
			Client: fb,
		})
	}
	return r, nil
}

// NewFromProviders builds a router from explicitly named providers.
func NewFromProviders(providers ...Provider) (*Router, error) {
	var list []Provider
	for _, p := range providers {
		if p.Client == nil {
			continue
		}
		if p.Name == "" {
			p.Name = "provider"
		}
		list = append(list, p)
	}
	if len(list) == 0 {
		return nil, llm.ErrNilClient
	}
	return &Router{providers: list}, nil
}

// Providers returns a copy of the configured provider list.
func (r *Router) Providers() []Provider {
	out := make([]Provider, len(r.providers))
	copy(out, r.providers)
	return out
}

// Chat tries each provider until one returns a successful Generation.
func (r *Router) Chat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(r.providers) == 0 {
		return nil, llm.ErrNilClient
	}

	var last error
	for _, p := range r.providers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		gen, err := p.Client.Chat(ctx, messages, opts...)
		if err == nil {
			return gen, nil
		}
		last = err
	}
	if last == nil {
		last = llm.ErrProvider
	}
	return nil, errors.Unavailable(ErrAllProvidersFailed.Message, last)
}

// StreamChat tries each provider until StreamChat succeeds (channel returned).
// Mid-stream errors are not retried on another provider.
func (r *Router) StreamChat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (<-chan llm.GenerationChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(r.providers) == 0 {
		return nil, llm.ErrNilClient
	}

	var last error
	for _, p := range r.providers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ch, err := p.Client.StreamChat(ctx, messages, opts...)
		if err == nil {
			return ch, nil
		}
		last = err
	}
	if last == nil {
		last = llm.ErrProvider
	}
	return nil, errors.Unavailable(ErrAllProvidersFailed.Message, last)
}

var _ llm.Client = (*Router)(nil)
