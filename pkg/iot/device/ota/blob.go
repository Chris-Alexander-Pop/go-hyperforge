package ota

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage/blob"
)

// Ensure BlobUpdater implements iot.Updater.
var _ iot.Updater = (*BlobUpdater)(nil)

// BlobConfig configures blob-backed OTA downloads.
type BlobConfig struct {
	// ManifestKey is the blob key for the JSON update manifest.
	ManifestKey string

	// MaxRetries for blob downloads (pkg/resilience).
	MaxRetries int
}

// BlobUpdater implements iot.Updater using pkg/storage/blob.Store for firmware.
//
// UpdateFile.URL is treated as a blob object key (optionally prefixed with "blob://").
type BlobUpdater struct {
	store    blob.Store
	cfg      BlobConfig
	progress ProgressCallback
	state    UpdateState
	retrier  resilience.Retrier
}

// NewBlobUpdater creates an OTA updater backed by store.
func NewBlobUpdater(store blob.Store, cfg BlobConfig) (*BlobUpdater, error) {
	if store == nil {
		return nil, iot.ErrInvalidConfig("blob store is required", nil)
	}
	if cfg.ManifestKey == "" {
		cfg.ManifestKey = "ota/manifest.json"
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	retryCfg := resilience.DefaultRetryConfig()
	retryCfg.MaxAttempts = cfg.MaxRetries
	retryCfg.InitialBackoff = 20 * time.Millisecond
	retryCfg.MaxBackoff = time.Second
	return &BlobUpdater{
		store:   store,
		cfg:     cfg,
		state:   StateIdle,
		retrier: resilience.NewRetrier(retryCfg),
	}, nil
}

// SetProgressCallback registers a progress reporter.
func (u *BlobUpdater) SetProgressCallback(cb ProgressCallback) {
	u.progress = cb
}

func (u *BlobUpdater) reportProgress(p UpdateProgress) {
	u.state = p.State
	if u.progress != nil {
		u.progress(p)
	}
}

// CheckForUpdate loads the manifest blob and compares versions.
func (u *BlobUpdater) CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateManifest, bool, error) {
	u.reportProgress(UpdateProgress{State: StateChecking})

	rc, err := u.store.Download(ctx, u.cfg.ManifestKey)
	if err != nil {
		u.reportProgress(UpdateProgress{State: StateFailed, Error: err.Error()})
		return nil, false, iot.ErrManifestNotFound(err)
	}
	defer rc.Close()

	var manifest UpdateManifest
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, false, pkgerrors.Internal("failed to read manifest", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, false, pkgerrors.Internal("failed to parse manifest", err)
	}

	newer, err := iot.IsNewerVersion(manifest.Version, currentVersion)
	if err != nil {
		u.reportProgress(UpdateProgress{State: StateFailed, Error: err.Error()})
		return nil, false, err
	}
	u.reportProgress(UpdateProgress{State: StateIdle})
	return &manifest, newer, nil
}

// DownloadUpdate fetches each file from the blob store and verifies SHA-256.
func (u *BlobUpdater) DownloadUpdate(ctx context.Context, manifest *UpdateManifest) (map[string][]byte, error) {
	if manifest == nil {
		return nil, iot.ErrInvalidConfig("manifest is required", nil)
	}
	u.reportProgress(UpdateProgress{State: StateDownloading})

	var totalSize int64
	for _, f := range manifest.Files {
		totalSize += f.Size
	}

	files := make(map[string][]byte, len(manifest.Files))
	var downloaded int64
	for _, file := range manifest.Files {
		u.reportProgress(UpdateProgress{
			State:           StateDownloading,
			CurrentFile:     file.Name,
			BytesDownloaded: downloaded,
			TotalBytes:      totalSize,
			Percentage:      percent(downloaded, totalSize),
		})
		data, err := u.downloadBlob(ctx, file)
		if err != nil {
			u.reportProgress(UpdateProgress{State: StateFailed, Error: err.Error()})
			return nil, err
		}
		files[file.Name] = data
		downloaded += file.Size
	}
	u.reportProgress(UpdateProgress{
		State:           StateVerifying,
		BytesDownloaded: totalSize,
		TotalBytes:      totalSize,
		Percentage:      100,
	})
	return files, nil
}

func (u *BlobUpdater) downloadBlob(ctx context.Context, file UpdateFile) ([]byte, error) {
	key := strings.TrimPrefix(file.URL, "blob://")
	if key == "" {
		key = file.Name
	}
	var data []byte
	err := u.retrier.Execute(ctx, func(ctx context.Context) error {
		rc, err := u.store.Download(ctx, key)
		if err != nil {
			return err
		}
		defer rc.Close()
		body, err := io.ReadAll(rc)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		got := hex.EncodeToString(sum[:])
		if file.SHA256 != "" && got != file.SHA256 {
			return iot.ErrChecksumMismatch(file.Name, file.SHA256, got)
		}
		data = body
		return nil
	})
	if err != nil {
		return nil, iot.ErrDownloadFailed(file.Name, err)
	}
	return data, nil
}

// ApplyUpdate marks install complete (platform-specific apply is reserved).
func (u *BlobUpdater) ApplyUpdate(ctx context.Context, files map[string][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	u.reportProgress(UpdateProgress{State: StateInstalling})
	_ = files
	u.reportProgress(UpdateProgress{State: StateComplete})
	return nil
}

// CheckAndApply checks and applies when an update is available.
func (u *BlobUpdater) CheckAndApply(ctx context.Context, deviceID, currentVersion string) error {
	_ = deviceID
	manifest, available, err := u.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		return err
	}
	if !available {
		return nil
	}
	files, err := u.DownloadUpdate(ctx, manifest)
	if err != nil {
		return err
	}
	return u.ApplyUpdate(ctx, files)
}

// GetState returns the current update state.
func (u *BlobUpdater) GetState() UpdateState {
	return u.state
}
