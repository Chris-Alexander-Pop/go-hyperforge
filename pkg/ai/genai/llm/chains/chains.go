// Package chains provides LangChain-style processing chains.
package chains

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm/memory"
)

// Chain is a unit of work.
type Chain interface {
	Run(ctx context.Context, input map[string]string) (map[string]string, error)
}

// LLMChain runs a prompt through an LLM.
type LLMChain struct {
	Client llm.Client
	Prompt string // Simple template
	Memory memory.Memory
}

func NewLLMChain(client llm.Client, prompt string) *LLMChain {
	return &LLMChain{
		Client: client,
		Prompt: prompt,
	}
}

func (c *LLMChain) Run(ctx context.Context, input map[string]string) (map[string]string, error) {
	// Simple interpolation
	text := c.Prompt
	for k, v := range input {
		// In production use a real template engine
		// text = strings.ReplaceAll(text, "{{"+k+"}}", v)
		_ = k
		_ = v
	}

	// Add to memory if exists (not implemented for simplicity of example, usually prompts are constructed from history + template)

	msg := llm.Message{Role: llm.RoleUser, Content: text}

	resp, err := c.Client.Chat(ctx, []llm.Message{msg})
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"response": resp.Message.Content,
	}, nil
}

// SequentialChain runs chains in order.
type SequentialChain struct {
	Chains []Chain
}

func (c *SequentialChain) Run(ctx context.Context, input map[string]string) (map[string]string, error) {
	currentResult := input
	for _, chain := range c.Chains {
		res, err := chain.Run(ctx, currentResult)
		if err != nil {
			return nil, err
		}
		// Merge results
		for k, v := range res {
			currentResult[k] = v
		}
	}
	return currentResult, nil
}
