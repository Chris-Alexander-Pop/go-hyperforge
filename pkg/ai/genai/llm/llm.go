// Package llm provides the core interfaces for Large Language Models.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
//
//	client, err := openai.New("key")
//	resp, err := client.Chat(ctx, []llm.Message{{Role: llm.RoleUser, Content: "Hello"}})
//
// For token streaming, use StreamChat (required on Client; adapters may buffer a full Chat
// response into a single chunk when native streaming is unavailable):
//
//	chunks, err := client.StreamChat(ctx, messages)
//
// Multimodal messages use Parts (text + image) while keeping Content for plain-text turns:
//
//	msg := llm.Message{Role: llm.RoleUser, Parts: []llm.ContentPart{
//		llm.TextPart("What is in this image?"),
//		llm.ImageURLPart("https://example.com/photo.png"),
//	}}
package llm

import (
	"context"
	"strings"
)

// Role defines who sent the message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleFunction  Role = "function"
	RoleTool      Role = "tool"
)

// PartType identifies a multimodal content part.
type PartType string

const (
	PartTypeText        PartType = "text"
	PartTypeImageURL    PartType = "image_url"
	PartTypeImageBase64 PartType = "image_base64"
)

// ContentPart is one piece of a multimodal message (text or image).
type ContentPart struct {
	Type     PartType `json:"type"`
	Text     string   `json:"text,omitempty"`
	ImageURL string   `json:"image_url,omitempty"` // remote URL or data: URI
	MIMEType string   `json:"mime_type,omitempty"` // e.g. image/png for base64
	Data     []byte   `json:"data,omitempty"`      // raw bytes for image_base64
}

// TextPart builds a text content part.
func TextPart(text string) ContentPart {
	return ContentPart{Type: PartTypeText, Text: text}
}

// ImageURLPart builds an image_url content part.
func ImageURLPart(url string) ContentPart {
	return ContentPart{Type: PartTypeImageURL, ImageURL: url}
}

// ImageBase64Part builds an inline image content part.
func ImageBase64Part(mime string, data []byte) ContentPart {
	return ContentPart{Type: PartTypeImageBase64, MIMEType: mime, Data: data}
}

// Message represents a single turn in a conversation.
// Prefer Content for plain text; set Parts for multimodal (text + images).
// When Parts is non-empty, adapters should prefer Parts over Content.
type Message struct {
	Role       Role                   `json:"role"`
	Content    string                 `json:"content,omitempty"`
	Parts      []ContentPart          `json:"parts,omitempty"`
	Name       string                 `json:"name,omitempty"` // For function/tool calls
	ToolCalls  []ToolCall             `json:"tool_calls,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"` // For tool responses
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// TextContent returns concatenated text from Content or text Parts.
func (m Message) TextContent() string {
	if m.Content != "" {
		return m.Content
	}
	var b strings.Builder
	for _, p := range m.Parts {
		if p.Type == PartTypeText && p.Text != "" {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// HasImages reports whether the message includes image parts.
func (m Message) HasImages() bool {
	for _, p := range m.Parts {
		if p.Type == PartTypeImageURL || p.Type == PartTypeImageBase64 {
			return true
		}
	}
	return false
}

// IsMultimodal reports whether the message uses content parts.
func (m Message) IsMultimodal() bool {
	return len(m.Parts) > 0
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
