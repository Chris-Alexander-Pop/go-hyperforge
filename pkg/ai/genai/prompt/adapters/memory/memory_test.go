package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/prompt/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func TestPromptVersioning(t *testing.T) {
	ctx := context.Background()
	s := memory.New()

	if err := s.Put(ctx, prompt.Template{Name: "greet", Version: "v1", Body: "Hello {{name}}"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(ctx, prompt.Template{Name: "greet", Version: "v2", Body: "Hi {{name}}!"}); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get(ctx, "greet", "v1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "Hello {{name}}" {
		t.Fatalf("v1 body=%q", got.Body)
	}

	latest, err := s.Get(ctx, "greet", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != "v2" {
		t.Fatalf("latest=%q", latest.Version)
	}

	out, err := s.Render(ctx, "greet", "v1", map[string]string{"name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello Ada" {
		t.Fatalf("render=%q", out)
	}

	_, err = s.Get(ctx, "missing", "")
	if !errors.Is(err, prompt.ErrNotFound) && !errors.IsCode(err, errors.CodeNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestPromptInvalid(t *testing.T) {
	err := memory.New().Put(context.Background(), prompt.Template{Name: "x", Version: "latest", Body: "y"})
	if err == nil {
		t.Fatal("expected invalid")
	}
}
