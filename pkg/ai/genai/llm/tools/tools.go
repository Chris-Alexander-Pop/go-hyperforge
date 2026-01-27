// Package tools provides a registry for agent tools.
package tools

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
)

// ToolFunc is the function signature for a tool.
type ToolFunc func(ctx context.Context, args []byte) (string, error)

// Registry manages available tools.
type Registry struct {
	tools map[string]RegisteredTool
}

type RegisteredTool struct {
	Def  llm.Tool
	Func ToolFunc
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]RegisteredTool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(name, description string, params interface{}, fn ToolFunc) {
	toolDef := llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  params,
		},
	}
	r.tools[name] = RegisteredTool{
		Def:  toolDef,
		Func: fn,
	}
}

// GetDefinitions returns the tool definitions for the LLM.
func (r *Registry) GetDefinitions() []llm.Tool {
	defs := make([]llm.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Def)
	}
	return defs
}

// Execute runs a named tool with arguments.
func (r *Registry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	return tool.Func(ctx, []byte(argsJSON))
}

// Helper to generate JSON schema (simplified)
func GenerateSchema(v interface{}) map[string]interface{} {
	// In a real implementation this would use reflection to generate JSON schema
	// For now returns the input as is, assuming user provided schema manually
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}
