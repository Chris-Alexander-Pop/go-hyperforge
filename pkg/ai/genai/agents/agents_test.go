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

func TestParseAction(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAction string
		wantInput  string
	}{
		{
			name:       "valid action",
			input:      "ACTION: Search INPUT: golang",
			wantAction: "Search",
			wantInput:  "golang",
		},
		{
			name:       "valid action with whitespace",
			input:      "ACTION:  Search   INPUT:   golang   ",
			wantAction: "Search",
			wantInput:  "golang",
		},
		{
			name:       "valid action with extra text",
			input:      "THOUGHT: reason\nACTION: Search INPUT: golang",
			wantAction: "Search",
			wantInput:  "golang",
		},
		{
			name:       "no match",
			input:      "no action here",
			wantAction: "",
			wantInput:  "",
		},
		{
			name:       "malformed action",
			input:      "ACTION: Search NOINPUT: golang",
			wantAction: "",
			wantInput:  "",
		},
	}

	agent := &Agent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAction, gotInput := agent.parseAction(tt.input)
			if gotAction != tt.wantAction {
				t.Errorf("parseAction() action = %v, want %v", gotAction, tt.wantAction)
			}
			if gotInput != tt.wantInput {
				t.Errorf("parseAction() input = %v, want %v", gotInput, tt.wantInput)
			}
		})
	}
}
