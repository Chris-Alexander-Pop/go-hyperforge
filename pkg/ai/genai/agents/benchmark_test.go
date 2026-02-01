package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
)

type MockTool struct {
	name        string
	description string
}

func (m MockTool) Name() string {
	return m.name
}

func (m MockTool) Description() string {
	return m.description
}

func (m MockTool) Run(ctx context.Context, input string) (string, error) {
	return "mock output", nil
}

type MockClient struct{}

func (m MockClient) Chat(ctx context.Context, history []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	return &llm.Generation{
		Message: llm.Message{
			Role:    llm.RoleAssistant,
			Content: "mock response",
		},
	}, nil
}

func BenchmarkBuildSystemPrompt(b *testing.B) {
	numTools := 100
	tools := make([]Tool, numTools)
	for i := 0; i < numTools; i++ {
		tools[i] = MockTool{
			name:        fmt.Sprintf("tool_%d", i),
			description: fmt.Sprintf("description for tool %d", i),
		}
	}

	agent := New(MockClient{}, tools)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agent.buildSystemPrompt()
	}
}
