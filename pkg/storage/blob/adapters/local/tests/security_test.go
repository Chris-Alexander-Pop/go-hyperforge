package local_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/adapters/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathTraversalProtection(t *testing.T) {
	// Setup
	storeDir := t.TempDir()

	// Create a "secret" file outside the store directory
	secretDir := t.TempDir()
	secretFile := filepath.Join(secretDir, "secret.txt")
	err := os.WriteFile(secretFile, []byte("super_secret_data"), 0644)
	require.NoError(t, err)

	// Initialize the store
	cfg := blob.Config{LocalDir: storeDir}
	store, err := local.New(cfg)
	require.NoError(t, err)

	// Construct a relative path that traverses out of the store directory
	relPath, err := filepath.Rel(storeDir, secretFile)
	require.NoError(t, err)

	// Ensure the path actually contains traversal characters
	// If the temp dirs are side-by-side, it will be "../<secretDir>/secret.txt"
	if !strings.Contains(relPath, "..") {
		// Force a traversal path if the OS/setup didn't give one naturally
		relPath = "../" + filepath.Base(secretDir) + "/secret.txt"
	}

	ctx := context.Background()

	t.Run("Download blocks traversal", func(t *testing.T) {
		_, err := store.Download(ctx, relPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal detected")
	})

	t.Run("Upload blocks traversal", func(t *testing.T) {
		err := store.Upload(ctx, relPath, strings.NewReader("malicious"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal detected")
	})

	t.Run("Delete blocks traversal", func(t *testing.T) {
		err := store.Delete(ctx, relPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal detected")
	})
}

func TestRelativePathInitialization(t *testing.T) {
	// This test ensures that initializing with a relative path works (e.g. ".")
	// and that traversal checks still work.

	// We create a temp dir, change cwd to it, and initialize store with "."
	tmpDir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(cwd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg := blob.Config{LocalDir: "."}
	store, err := local.New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Test that valid access works
	err = store.Upload(ctx, "test.txt", strings.NewReader("content"))
	require.NoError(t, err)

	// Test that traversal is blocked
	// Since we are at CWD, any ".." attempts to go to parent of temp dir, which is allowed by OS but should be blocked by Store
	// Wait, if baseDir is abs(CWD), then ".." goes to parent.

	_, err = store.Download(ctx, "../outside.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal detected")
}
