package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/template"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/template/adapters/html"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/template/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/template/adapters/text"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type TemplateTestSuite struct {
	test.Suite
	engine template.Engine
}

func (s *TemplateTestSuite) SetupTest() {
	s.Suite.SetupTest()
	mem := memory.New()
	s.NoError(mem.AddTemplate("welcome", "Hello {{.Name}}"))
	s.engine = mem
}

func (s *TemplateTestSuite) TestRender() {
	result, err := s.engine.Render(s.Ctx, "welcome", map[string]string{"Name": "World"})
	s.NoError(err)
	s.Equal("Hello World", result)
}

func (s *TemplateTestSuite) TestRenderNotFound() {
	_, err := s.engine.Render(s.Ctx, "missing", nil)
	s.Error(err)
}

func TestTemplateSuite(t *testing.T) {
	test.Run(t, new(TemplateTestSuite))
}

func TestTextEngineFromDir(t *testing.T) {
	dir := t.TempDir()
	requireWrite(t, filepath.Join(dir, "hello.tmpl"), "Hello {{.Name}}")

	engine, err := text.New(template.Config{Driver: "text", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	out, err := engine.Render(context.Background(), "hello", map[string]string{"Name": "Go"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello Go" {
		t.Fatalf("got %q", out)
	}
}

func TestHTMLEngineFromDir(t *testing.T) {
	dir := t.TempDir()
	requireWrite(t, filepath.Join(dir, "page.html"), "<h1>{{.Title}}</h1>")

	engine, err := html.New(template.Config{Driver: "html", Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	out, err := engine.Render(context.Background(), "page", map[string]string{"Title": "Hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<h1>Hi</h1>" {
		t.Fatalf("got %q", out)
	}
}

func requireWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
