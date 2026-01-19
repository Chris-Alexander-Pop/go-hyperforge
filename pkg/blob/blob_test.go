package blob_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/blob/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestBlobStore(t *testing.T) {
	// Use In-Memory adapter for fast testing
	store := memory.New(blob.Config{})
	instrumented := blob.NewInstrumentedStore(store, "test-blob")

	ctx := context.Background()
	key := "test.txt"
	content := "hello world"

	// 1. Upload
	err := instrumented.Upload(ctx, key, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// 2. Download
	rc, err := instrumented.Download(ctx, key)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected %s, got %s", content, string(data))
	}

	// 3. Delete
	err = instrumented.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 4. Verify Gone
	_, err = instrumented.Download(ctx, key)
	if err == nil {
		t.Error("Expected error after delete, got nil")
	} else {
		// Ensure it's a NotFound error
		var appErr *errors.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != errors.CodeNotFound {
				t.Errorf("Expected NotFound code, got %s", appErr.Code)
			}
		}
	}
}
