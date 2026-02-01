// Package agents provides a simple ReAct agent framework.
package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
)

// Agent runs autonomous tasks.
type Agent struct {
	client   llm.Client
	tools    map[string]Tool
	maxSteps int
}

// Tool is a function the agent can call.
type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, input string) (string, error)
}

// New creates a new agent.
func New(client llm.Client, tools []Tool) *Agent {
	t := make(map[string]Tool)
	for _, tool := range tools {
		t[tool.Name()] = tool
	}
	return &Agent{
		client:   client,
		tools:    t,
		maxSteps: 10,
	}
}

// Run executes a task using the ReAct loop.
func (a *Agent) Run(ctx context.Context, task string) (string, error) {
	prompt := a.buildSystemPrompt()
	history := []llm.Message{
		{Role: llm.RoleSystem, Content: prompt},
		{Role: llm.RoleUser, Content: task},
	}

	for i := 0; i < a.maxSteps; i++ {
		// Use functional option for Temperature
		resp, err := a.client.Chat(ctx, history, llm.WithTemperature(0))
		if err != nil {
			return "", err
		}

		content := resp.Message.Content
		history = append(history, resp.Message)

		if strings.Contains(content, "FINAL ANSWER:") {
			parts := strings.Split(content, "FINAL ANSWER:")
			return strings.TrimSpace(parts[1]), nil
		}

		// Parse Action
		action, input := a.parseAction(content)
		if action == "" {
			// No action, treat as observation needed or simple continuation
			continue
		}

		// Execute Tool
		tool, ok := a.tools[action]
		result := ""
		if !ok {
			result = fmt.Sprintf("Error: Tool %s not found", action)
		} else {
			res, err := tool.Run(ctx, input)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			} else {
				result = res
			}
		}

		// Observation
		observation := fmt.Sprintf("OBSERVATION: %s", result)
		history = append(history, llm.Message{Role: llm.RoleUser, Content: observation})
	}

	return "", fmt.Errorf("max steps reached")
}

func (a *Agent) buildSystemPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are an AI agent. Solve the user's task using the available tools.\nAvailable Tools:\n")

	for _, t := range a.tools {
		fmt.Fprintf(&sb, "- %s: %s\n", t.Name(), t.Description())
	}

	sb.WriteString(`
Format your response as:
THOUGHT: reason about what to do
ACTION: ToolName
INPUT: input for the tool

After getting an OBSERVATION, repeat the cycle.
When you have the answer, output:
FINAL ANSWER: the answer
`)

	return sb.String()
}

func (a *Agent) parseAction(text string) (string, string) {
	re := regexp.MustCompile(`ACTION:\s*(\w+)\s*INPUT:\s*(.*)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) == 3 {
		return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
	}
	return "", ""
}
