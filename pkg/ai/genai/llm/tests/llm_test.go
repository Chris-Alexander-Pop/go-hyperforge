package llm_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/adapters/memory"
	llmmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func TestMemoryChatAndStream(t *testing.T) {
	ctx := context.Background()
	client := memory.New().WithResponse("hello", "Hello there, world!").WithChunkSize(5)

	gen, err := client.Chat(ctx, []llm.Message{{Role: llm.RoleUser, Content: "say hello"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(gen.Message.Content, "Hello") {
		t.Fatalf("unexpected content: %q", gen.Message.Content)
	}

	ch, err := client.StreamChat(ctx, []llm.Message{{Role: llm.RoleUser, Content: "say hello"}})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var built strings.Builder
	var finish string
	chunks := 0
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk err: %v", chunk.Err)
		}
		built.WriteString(chunk.Delta)
		chunks++
		if chunk.FinishReason != "" {
			finish = chunk.FinishReason
		}
	}
	if chunks < 2 {
		t.Fatalf("expected multi-chunk stream, got %d", chunks)
	}
	if built.String() != gen.Message.Content {
		t.Fatalf("stream=%q chat=%q", built.String(), gen.Message.Content)
	}
	if finish != "stop" {
		t.Fatalf("finish=%q", finish)
	}
}

func TestMemoryChatEmptyMessages(t *testing.T) {
	_, err := memory.New().Chat(context.Background(), nil)
	if !errors.IsCode(err, errors.CodeInvalidArgument) {
		t.Fatalf("want INVALID_ARGUMENT, got %v", err)
	}
}

func TestInstrumentedStreamChat(t *testing.T) {
	ctx := context.Background()
	inner := memory.New().WithResponse("ping", "pong-response")
	client := llm.NewInstrumentedClient(inner)

	ch, err := client.StreamChat(ctx, []llm.Message{{Role: llm.RoleUser, Content: "ping"}})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	var got string
	for chunk := range ch {
		got += chunk.Delta
	}
	if got != "pong-response" {
		t.Fatalf("got %q", got)
	}
}

func TestStreamFromChat(t *testing.T) {
	ctx := context.Background()
	chat := func(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
		return &llm.Generation{
			Message:      llm.Message{Role: llm.RoleAssistant, Content: "one-shot"},
			FinishReason: "stop",
			Usage:        llm.Usage{TotalTokens: 3},
		}, nil
	}
	ch, err := llm.StreamFromChat(ctx, chat, []llm.Message{{Role: llm.RoleUser, Content: "x"}})
	if err != nil {
		t.Fatalf("StreamFromChat: %v", err)
	}
	n := 0
	for chunk := range ch {
		n++
		if chunk.Delta != "one-shot" {
			t.Fatalf("delta=%q", chunk.Delta)
		}
	}
	if n != 1 {
		t.Fatalf("chunks=%d", n)
	}
}

func TestConversationMemoryContext(t *testing.T) {
	ctx := context.Background()
	mem := llmmemory.NewSimpleMemory(2)

	if err := mem.AddUserMessage(ctx, "hi"); err != nil {
		t.Fatal(err)
	}
	if err := mem.AddAssistantMessage(ctx, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := mem.AddUserMessage(ctx, "again"); err != nil {
		t.Fatal(err)
	}

	msgs, err := mem.GetMessages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("maxLen truncate: got %d", len(msgs))
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := mem.AddMessage(canceled, llm.Message{Role: llm.RoleUser, Content: "nope"}); err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestStreamChatCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := memory.New().StreamChat(ctx, []llm.Message{{Role: llm.RoleUser, Content: "x"}})
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestStreamChatDoesNotHang(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx := context.Background()
		ch, err := memory.New().WithChunkSize(3).StreamChat(ctx, []llm.Message{
			{Role: llm.RoleUser, Content: "hello streaming"},
		})
		if err != nil {
			t.Errorf("StreamChat: %v", err)
			return
		}
		for range ch {
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StreamChat hung")
	}
}
