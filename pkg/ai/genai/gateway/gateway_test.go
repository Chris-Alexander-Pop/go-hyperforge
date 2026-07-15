package gateway_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/gateway"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// failingClient always fails Chat/StreamChat.
type failingClient struct{}

func (f *failingClient) Chat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (*llm.Generation, error) {
	return nil, llm.WrapProvider(errors.Unavailable("primary down", nil))
}

func (f *failingClient) StreamChat(ctx context.Context, messages []llm.Message, opts ...llm.GenerateOption) (<-chan llm.GenerationChunk, error) {
	return nil, llm.WrapProvider(errors.Unavailable("primary down", nil))
}

func TestGatewayFallbackChat(t *testing.T) {
	primary := &failingClient{}
	fallback := memory.New().WithResponse("hello", "from-fallback")

	r, err := gateway.New(primary, fallback)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(r.Providers()) != 2 {
		t.Fatalf("providers=%d", len(r.Providers()))
	}

	gen, err := r.Chat(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "say hello"},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(gen.Message.Content, "from-fallback") {
		t.Fatalf("got %q", gen.Message.Content)
	}
}

func TestGatewayFallbackStream(t *testing.T) {
	r, err := gateway.New(&failingClient{}, memory.New().WithResponse("ping", "pong").WithChunkSize(2))
	if err != nil {
		t.Fatal(err)
	}
	ch, err := r.StreamChat(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "ping"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for c := range ch {
		if c.Err != nil {
			t.Fatal(c.Err)
		}
		got.WriteString(c.Delta)
	}
	if got.String() != "pong" {
		t.Fatalf("got %q", got.String())
	}
}

func TestGatewayAllFail(t *testing.T) {
	r, err := gateway.New(&failingClient{}, &failingClient{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Chat(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: "x"}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.IsCode(err, errors.CodeUnavailable) {
		t.Fatalf("want UNAVAILABLE, got %v", err)
	}
}

func TestGatewayNilPrimary(t *testing.T) {
	_, err := gateway.New(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGatewayPrimarySuccess(t *testing.T) {
	primary := memory.New().WithResponse("hi", "primary-ok")
	fallback := memory.New().WithResponse("hi", "fallback-ok")
	r, err := gateway.New(primary, fallback)
	if err != nil {
		t.Fatal(err)
	}
	gen, err := r.Chat(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if gen.Message.Content != "primary-ok" {
		t.Fatalf("got %q", gen.Message.Content)
	}
}

func TestNewFromProviders(t *testing.T) {
	r, err := gateway.NewFromProviders(
		gateway.Provider{Name: "a", Client: &failingClient{}},
		gateway.Provider{Name: "b", Client: memory.New().WithResponse("z", "ok")},
	)
	if err != nil {
		t.Fatal(err)
	}
	gen, err := r.Chat(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: "z"}})
	if err != nil {
		t.Fatal(err)
	}
	if gen.Message.Content != "ok" {
		t.Fatalf("got %q", gen.Message.Content)
	}
}
