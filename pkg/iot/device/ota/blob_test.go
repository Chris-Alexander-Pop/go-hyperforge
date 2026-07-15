package ota_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/ota"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
	blobmem "github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlobUpdater_DownloadAndApply(t *testing.T) {
	store := blobmem.New(blob.Config{})
	ctx := context.Background()

	firmware := []byte("firmware-v2-payload")
	sum := sha256.Sum256(firmware)
	checksum := hex.EncodeToString(sum[:])

	manifest := iot.UpdateManifest{
		Version:     "v2.0.0",
		Description: "test",
		ReleaseDate: time.Now().UTC(),
		Files: []iot.UpdateFile{{
			Name:   "app.bin",
			URL:    "blob://firmware/app.bin",
			Size:   int64(len(firmware)),
			SHA256: checksum,
		}},
	}
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, store.Upload(ctx, "ota/manifest.json", bytes.NewReader(raw)))
	require.NoError(t, store.Upload(ctx, "firmware/app.bin", bytes.NewReader(firmware)))

	updater, err := ota.NewBlobUpdater(store, ota.BlobConfig{ManifestKey: "ota/manifest.json"})
	require.NoError(t, err)

	m, newer, err := updater.CheckForUpdate(ctx, "v1.0.0")
	require.NoError(t, err)
	assert.True(t, newer)
	assert.Equal(t, "v2.0.0", m.Version)

	files, err := updater.DownloadUpdate(ctx, m)
	require.NoError(t, err)
	assert.Equal(t, firmware, files["app.bin"])

	require.NoError(t, updater.ApplyUpdate(ctx, files))
	assert.Equal(t, iot.StateComplete, updater.GetState())

	require.NoError(t, updater.CheckAndApply(ctx, "dev-1", "v1.0.0"))
}

func TestBlobUpdater_NilStore(t *testing.T) {
	_, err := ota.NewBlobUpdater(nil, ota.BlobConfig{})
	require.Error(t, err)
}
