package local_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob/adapters/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathTraversal(t *testing.T) {
	// Create a temp directory for our test
	tempDir, err := os.MkdirTemp("", "blob-security-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a "secret" file outside the blob storage root
	secretFile := filepath.Join(tempDir, "secret.txt")
	err = os.WriteFile(secretFile, []byte("super secret data"), 0644)
	require.NoError(t, err)

	// Create the blob storage root inside the temp dir
	blobDir := filepath.Join(tempDir, "blobs")
	err = os.Mkdir(blobDir, 0755)
	require.NoError(t, err)

	// Initialize the store
	store, err := local.New(blob.Config{LocalDir: blobDir})
	require.NoError(t, err)

	// Try to access the secret file using path traversal
	// "../secret.txt" should work relative to "blobs"
	traversalKey := "../secret.txt"

	// 1. Test Download - Verify Protection
	t.Run("Download Path Traversal Protection", func(t *testing.T) {
		_, err := store.Download(context.Background(), traversalKey)
		require.Error(t, err, "Should reject path traversal")
		assert.Contains(t, err.Error(), "path traversal")
	})

	// 2. Test Delete - Verify Protection
	// We create another file to delete
	deleteTarget := filepath.Join(tempDir, "delete_me.txt")
	err = os.WriteFile(deleteTarget, []byte("delete me"), 0644)
	require.NoError(t, err)

	t.Run("Delete Path Traversal Protection", func(t *testing.T) {
		err := store.Delete(context.Background(), "../delete_me.txt")
		require.Error(t, err, "Should reject path traversal")
		assert.Contains(t, err.Error(), "path traversal")

		// Verify file still exists
		_, err = os.Stat(deleteTarget)
		assert.NoError(t, err, "Target file should not be deleted")
	})

	// 3. Test Upload - Verify Protection
	t.Run("Upload Path Traversal Protection", func(t *testing.T) {
		// Try to overwrite secret file
		f, err := os.Open(secretFile) // just use existing file as source
		require.NoError(t, err)
		defer f.Close()

		err = store.Upload(context.Background(), "../secret.txt", f)
		require.Error(t, err, "Should reject path traversal")
		assert.Contains(t, err.Error(), "path traversal")
	})
}
