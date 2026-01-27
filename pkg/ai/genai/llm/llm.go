// Package llm provides the core interfaces for Large Language Models.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
//
//	client, err := openai.New("key")
//	resp, err := client.Generate(ctx, "Hello world")
package llm

import "context"

// Role defines who sent the message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleFunction  Role = "function"
	RoleTool      Role = "tool"
)

// Message represents a single turn in a conversation.
type Message struct {
	Role       Role                   `json:"role"`
	Content    string                 `json:"content"`
	Name       string                 `json:"name,omitempty"` // For function/tool calls
	ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"` // For tool responses
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a request to call a function.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // usually "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall details.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Generation represents the model's response.
type Generation struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // stop, length, tool_calls, content_filter
	Usage        Usage   `json:"usage"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// GenerateOptions configures the generation request.
type GenerateOptions struct {
	Model       string   `json:"model"`
	Temperature float64  `json:"temperature"`
	MaxTokens   int      `json:"max_tokens"`
	TopP        float64  `json:"top_p"`
	Stop        []string `json:"stop,omitempty"`
	Tools       []Tool   `json:"tools,omitempty"` // Available tools
}

// Tool definition for the model.
type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON Schema
}

// Client is the interface for LLM providers.
type Client interface {
	// Chat sends a conversation history and gets a response.
	Chat(ctx context.Context, messages []Message, opts ...GenerateOption) (*Generation, error)

	// StreamChat streams the response (implementation optional).
	// StreamChat(ctx context.Context, messages []Message, opts ...GenerateOption) (<-chan GenerationChunk, error)
}

// GenerateOption is a functional option.
type GenerateOption func(*GenerateOptions)

func WithModel(model string) GenerateOption {
	return func(o *GenerateOptions) { o.Model = model }
}

func WithTemperature(temp float64) GenerateOption {
	return func(o *GenerateOptions) { o.Temperature = temp }
}

func WithTools(tools []Tool) GenerateOption {
	return func(o *GenerateOptions) { o.Tools = tools }
}
