package agents

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	tools := []Tool{
		MockTool{name: "search", description: "search the web"},
		MockTool{name: "calc", description: "calculate stuff"},
	}
	agent := New(MockClient{}, tools)
	prompt := agent.buildSystemPrompt()

	// Check for correct formatting of tool list
	expectedParts := []string{
		"- search: search the web",
		"- calc: calculate stuff",
	}

	for _, part := range expectedParts {
		if !strings.Contains(prompt, part) {
			t.Errorf("Prompt missing expected string: %q\nGot:\n%s", part, prompt)
		}
	}
}
