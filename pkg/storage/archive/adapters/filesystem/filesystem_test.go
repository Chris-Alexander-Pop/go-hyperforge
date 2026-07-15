package filesystem

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/archive"
	"github.com/stretchr/testify/require"
)

func TestFilesystemArchive(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()

	data := []byte("cold-backup-bytes")
	require.NoError(t, store.Archive(ctx, "backups/db.dump", bytes.NewReader(data), archive.ArchiveOptions{
		Metadata: map[string]string{"src": "test"},
	}))

	obj, err := store.GetObject(ctx, "backups/db.dump")
	require.NoError(t, err)
	require.Equal(t, int64(len(data)), obj.Size)
	require.NotEmpty(t, obj.Checksum)

	_, err = store.Download(ctx, "backups/db.dump")
	require.Error(t, err)

	job, err := store.Restore(ctx, "backups/db.dump", archive.RestoreOptions{TTL: time.Hour})
	require.NoError(t, err)
	require.Equal(t, archive.RestoreStatusCompleted, job.Status)

	rc, err := store.Download(ctx, "backups/db.dump")
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	_ = rc.Close()
	require.NoError(t, err)
	require.Equal(t, data, got)

	list, err := store.List(ctx, archive.ListOptions{Prefix: "backups/"})
	require.NoError(t, err)
	require.Len(t, list.Objects, 1)

	require.NoError(t, store.Delete(ctx, "backups/db.dump"))
	_, err = store.GetObject(ctx, "backups/db.dump")
	require.Error(t, err)
}

func TestPathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	require.NoError(t, err)
	ctx := context.Background()

	// Logical ".." segments are cleaned; object must remain under root/objects.
	require.NoError(t, store.Archive(ctx, "a/../../outside", bytes.NewReader([]byte("x")), archive.ArchiveOptions{}))
	objPath := filepath.Join(root, "objects", "outside")
	_, err = os.Stat(objPath)
	require.NoError(t, err, "cleaned key should land under objects/")

	// Absolute-style keys also normalize under the cold root.
	require.NoError(t, store.Archive(ctx, "/abs/path", bytes.NewReader([]byte("y")), archive.ArchiveOptions{}))
	_, err = os.Stat(filepath.Join(root, "objects", "abs", "path"))
	require.NoError(t, err)
}
