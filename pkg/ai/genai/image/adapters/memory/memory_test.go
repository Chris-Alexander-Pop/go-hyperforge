package memory_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/image"
	imgmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/image/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func TestGenerate(t *testing.T) {
	svc := imgmemory.New("")
	urls, err := svc.Generate(context.Background(), "a cat", image.Options{N: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 2 {
		t.Fatalf("got %d urls", len(urls))
	}
	if !strings.Contains(urls[0], "cat") {
		t.Fatalf("url=%s", urls[0])
	}

	_, err = svc.Generate(context.Background(), "", image.Options{})
	if !errors.IsCode(err, errors.CodeInvalidArgument) {
		t.Fatalf("want invalid argument, got %v", err)
	}
}
