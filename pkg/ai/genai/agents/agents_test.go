package agents

import (
	"testing"
)

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
