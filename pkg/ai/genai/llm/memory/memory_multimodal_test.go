package memory_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm"
	llmmemory "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm/memory"
	adapter "github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm/adapters/memory"
)

func TestSimpleMemory_MultimodalParts(t *testing.T) {
	ctx := context.Background()
	mem := llmmemory.NewSimpleMemory(10)

	err := mem.AddUserParts(ctx,
		llm.TextPart("What animal is this?"),
		llm.ImageURLPart("https://example.com/cat.png"),
	)
	if err != nil {
		t.Fatalf("AddUserParts: %v", err)
	}

	msgs, err := mem.GetMessages(ctx)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	if !msgs[0].IsMultimodal() || !msgs[0].HasImages() {
		t.Fatalf("expected multimodal with image: %+v", msgs[0])
	}
	if msgs[0].TextContent() != "What animal is this?" {
		t.Fatalf("text=%q", msgs[0].TextContent())
	}
	if msgs[0].Content == "" {
		t.Fatal("expected Content mirrored from text parts")
	}
}

func TestMemoryAdapter_MultimodalChat(t *testing.T) {
	ctx := context.Background()
	client := adapter.New().WithResponse("animal", "It looks like a cat")

	msg := llm.Message{
		Role: llm.RoleUser,
		Parts: []llm.ContentPart{
			llm.TextPart("What animal is this?"),
			llm.ImageURLPart("https://cdn.example/cat.jpg"),
			llm.ImageBase64Part("image/png", []byte{0x89, 0x50, 0x4e, 0x47}),
		},
	}

	gen, err := client.Chat(ctx, []llm.Message{msg})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(gen.Message.Content, "cat") {
		t.Fatalf("content=%q", gen.Message.Content)
	}
	if !strings.Contains(gen.Message.Content, "[image:") {
		t.Fatalf("expected image acknowledgement, got %q", gen.Message.Content)
	}
	if !strings.Contains(gen.Message.Content, "[image_b64:") {
		t.Fatalf("expected base64 image note, got %q", gen.Message.Content)
	}
}

func TestSimpleMemory_RejectEmpty(t *testing.T) {
	err := llmmemory.NewSimpleMemory(0).AddMessage(context.Background(), llm.Message{Role: llm.RoleUser})
	if err == nil {
		t.Fatal("expected error")
	}
}
