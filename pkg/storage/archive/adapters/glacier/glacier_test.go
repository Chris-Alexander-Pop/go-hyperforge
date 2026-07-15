package glacier_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/archive"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/archive/adapters/glacier"
	"github.com/stretchr/testify/require"
)

func TestGlacierThinAdapter(t *testing.T) {
	api := glacier.NewMemoryObjectAPI()
	store, err := glacier.NewFromAPI(api, glacier.Config{
		Bucket:             "cold",
		StorageClass:       archive.StorageClassArchive,
		DefaultRestoreTier: archive.RestoreTierStandard,
		RestoreTTL:         time.Hour,
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.Archive(ctx, "backup/db.dump", bytes.NewReader([]byte("payload")), archive.ArchiveOptions{
		Metadata: map[string]string{"env": "test"},
	}))

	obj, err := store.GetObject(ctx, "backup/db.dump")
	require.NoError(t, err)
	require.Equal(t, int64(7), obj.Size)
	require.Equal(t, "test", obj.Metadata["env"])

	job, err := store.Restore(ctx, "backup/db.dump", archive.RestoreOptions{Tier: archive.RestoreTierStandard})
	require.NoError(t, err)
	require.Equal(t, archive.RestoreStatusInProgress, job.Status)

	_, err = store.Download(ctx, "backup/db.dump")
	require.ErrorIs(t, err, archive.ErrObjectNotRestored)

	require.NoError(t, store.CompleteRestore("backup/db.dump"))
	status, err := store.GetRestoreStatus(ctx, "backup/db.dump")
	require.NoError(t, err)
	require.Equal(t, archive.RestoreStatusCompleted, status.Status)

	rc, err := store.Download(ctx, "backup/db.dump")
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	_ = rc.Close()
	require.NoError(t, err)
	require.Equal(t, "payload", string(data))

	// Expedited completes immediately.
	require.NoError(t, store.Archive(ctx, "backup/fast.dump", bytes.NewReader([]byte("x")), archive.ArchiveOptions{}))
	fast, err := store.Restore(ctx, "backup/fast.dump", archive.RestoreOptions{Tier: archive.RestoreTierExpedited})
	require.NoError(t, err)
	require.Equal(t, archive.RestoreStatusCompleted, fast.Status)

	list, err := store.List(ctx, archive.ListOptions{Prefix: "backup/"})
	require.NoError(t, err)
	require.Len(t, list.Objects, 2)

	require.NoError(t, store.Delete(ctx, "backup/db.dump"))
	_, err = store.GetObject(ctx, "backup/db.dump")
	require.ErrorIs(t, err, archive.ErrObjectNotFound)
}
