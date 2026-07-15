// Package llm provides the core interfaces for Large Language Models.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
//
//	client, err := openai.New("key")
//	resp, err := client.Chat(ctx, []llm.Message{{Role: llm.RoleUser, Content: "Hello"}})
//
// For token streaming, use StreamChat (required on Client; adapters may buffer a full Chat
// response into a single chunk when native streaming is unavailable):
//
//	chunks, err := client.StreamChat(ctx, messages)
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

// GenerationChunk is one piece of a streamed Chat response.
type GenerationChunk struct {
	// Delta is the incremental assistant text for this chunk.
	Delta string `json:"delta,omitempty"`
	// FinishReason is set on the final chunk (stop, length, tool_calls, …).
	FinishReason string `json:"finish_reason,omitempty"`
	// Usage is optionally set on the final chunk.
	Usage *Usage `json:"usage,omitempty"`
	// Err is a terminal stream error (channel closes after an error chunk).
	Err error `json:"-"`
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
// Chat returns a complete response; StreamChat yields incremental GenerationChunks.
type Client interface {
	// Chat sends a conversation history and gets a complete response.
	Chat(ctx context.Context, messages []Message, opts ...GenerateOption) (*Generation, error)

	// StreamChat streams the assistant response as GenerationChunks.
	// The returned channel is closed when the stream ends (or after an error chunk).
	StreamChat(ctx context.Context, messages []Message, opts ...GenerateOption) (<-chan GenerationChunk, error)
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

func WithMaxTokens(n int) GenerateOption {
	return func(o *GenerateOptions) { o.MaxTokens = n }
}

// ApplyOptions builds GenerateOptions from functional options.
func ApplyOptions(opts ...GenerateOption) GenerateOptions {
	o := GenerateOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

// StreamFromChat adapts a non-streaming Chat call into a single-chunk StreamChat channel.
// Useful for cloud adapters that have not yet wired native HTTP/SSE streaming.
func StreamFromChat(ctx context.Context, chat func(context.Context, []Message, ...GenerateOption) (*Generation, error), messages []Message, opts ...GenerateOption) (<-chan GenerationChunk, error) {
	if chat == nil {
		return nil, ErrNilClient
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, ErrEmptyMessages
	}

	ch := make(chan GenerationChunk, 1)
	go func() {
		defer close(ch)
		gen, err := chat(ctx, messages, opts...)
		if err != nil {
			select {
			case ch <- GenerationChunk{Err: err}:
			case <-ctx.Done():
			}
			return
		}
		usage := gen.Usage
		select {
		case ch <- GenerationChunk{
			Delta:        gen.Message.Content,
			FinishReason: gen.FinishReason,
			Usage:        &usage,
		}:
		case <-ctx.Done():
		}
	}()
	return ch, nil
}
