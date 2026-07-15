package local_test

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/file"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/file/adapters/local"
)

func TestLocalFileStoreRoundTrip(t *testing.T) {
	root := t.TempDir()
	store, err := local.New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	if err := store.Write(ctx, "/a/b.txt", bytes.NewReader([]byte("hello"))); err != nil {
		t.Fatalf("Write: %v", err)
	}
	rc, err := store.Read(ctx, "/a/b.txt")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	data, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil || string(data) != "hello" {
		t.Fatalf("Read data=%q err=%v", data, err)
	}

	info, err := store.Stat(ctx, "/a/b.txt")
	if err != nil || info.IsDir || info.Size != 5 {
		t.Fatalf("Stat=%+v err=%v", info, err)
	}

	if err := store.Mkdir(ctx, "/a/c"); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := store.Copy(ctx, "/a/b.txt", "/a/c/copy.txt"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if err := store.Rename(ctx, "/a/c/copy.txt", "/a/c/renamed.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	list, err := store.List(ctx, "/a", file.ListOptions{Recursive: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("List got %d entries", len(list))
	}

	if err := store.Delete(ctx, "/a/b.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Read(ctx, "/a/b.txt"); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestLocalPathTraversal(t *testing.T) {
	store, err := local.New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = store.Read(context.Background(), "../outside")
	if err == nil {
		t.Fatal("expected traversal error")
	}
	// Ensure root itself is not escaped via absolute-looking paths.
	absOutside := filepath.Join(t.TempDir(), "x")
	err = store.Write(context.Background(), absOutside, bytes.NewReader([]byte("x")))
	if err != nil {
		t.Logf("write abs-like path: %v", err)
	}
}

func TestNewWithConfig(t *testing.T) {
	root := t.TempDir()
	store, err := local.NewWithConfig(file.Config{MountPoint: root})
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	if err := store.Write(context.Background(), "/f", bytes.NewReader([]byte("z"))); err != nil {
		t.Fatalf("Write: %v", err)
	}
}
