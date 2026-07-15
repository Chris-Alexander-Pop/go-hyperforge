package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/iot"
)

// Ensure compile-time interface compliance.
var _ iot.Updater = (*Updater)(nil)

// UpdaterConfig configures the in-memory OTA updater.
type UpdaterConfig struct {
	// Manifest is the update manifest served by CheckForUpdate.
	// If nil, CheckForUpdate returns ErrManifestNotFound.
	Manifest *iot.UpdateManifest

	// Files maps file name → payload bytes for DownloadUpdate.
	// SHA-256 is verified against Manifest.Files[].SHA256.
	Files map[string][]byte

	// ApplyFn optionally overrides ApplyUpdate behavior.
	// When nil, ApplyUpdate marks state complete and succeeds.
	ApplyFn func(ctx context.Context, files map[string][]byte) error
}

// Updater is an in-memory OTA updater for tests.
type Updater struct {
	mu       *concurrency.SmartRWMutex
	cfg      UpdaterConfig
	progress iot.ProgressCallback
	state    atomic.Value // iot.UpdateState
}

// NewUpdater creates an in-memory OTA updater.
func NewUpdater(cfg UpdaterConfig) *Updater {
	if cfg.Files == nil {
		cfg.Files = make(map[string][]byte)
	}
	u := &Updater{
		mu:  concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "iot-memory-ota"}),
		cfg: cfg,
	}
	u.state.Store(iot.StateIdle)
	return u
}

// SetManifest replaces the served manifest (test helper).
func (u *Updater) SetManifest(m *iot.UpdateManifest) {
	u.mu.Lock()
	u.cfg.Manifest = m
	u.mu.Unlock()
}

// SetFile stores a downloadable file payload (test helper).
func (u *Updater) SetFile(name string, data []byte) {
	u.mu.Lock()
	u.cfg.Files[name] = append([]byte(nil), data...)
	u.mu.Unlock()
}

// SetProgressCallback registers a progress reporter.
func (u *Updater) SetProgressCallback(cb iot.ProgressCallback) {
	u.mu.Lock()
	u.progress = cb
	u.mu.Unlock()
}

func (u *Updater) reportProgress(p iot.UpdateProgress) {
	u.state.Store(p.State)
	u.mu.RLock()
	cb := u.progress
	u.mu.RUnlock()
	if cb != nil {
		cb(p)
	}
}

// CheckForUpdate returns the configured manifest when newer than currentVersion.
func (u *Updater) CheckForUpdate(ctx context.Context, currentVersion string) (*iot.UpdateManifest, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	u.reportProgress(iot.UpdateProgress{State: iot.StateChecking})

	u.mu.RLock()
	manifest := u.cfg.Manifest
	u.mu.RUnlock()

	if manifest == nil {
		u.reportProgress(iot.UpdateProgress{State: iot.StateFailed, Error: "manifest not found"})
		return nil, false, iot.ErrManifestNotFound(nil)
	}

	newer, err := iot.IsNewerVersion(manifest.Version, currentVersion)
	if err != nil {
		u.reportProgress(iot.UpdateProgress{State: iot.StateFailed, Error: err.Error()})
		return nil, false, err
	}

	u.reportProgress(iot.UpdateProgress{State: iot.StateIdle})
	return cloneManifest(manifest), newer, nil
}

// DownloadUpdate returns configured file payloads after SHA-256 verification.
func (u *Updater) DownloadUpdate(ctx context.Context, manifest *iot.UpdateManifest) (map[string][]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, iot.ErrInvalidConfig("manifest is required", nil)
	}

	u.reportProgress(iot.UpdateProgress{State: iot.StateDownloading})

	var totalSize int64
	for _, f := range manifest.Files {
		totalSize += f.Size
	}

	out := make(map[string][]byte, len(manifest.Files))
	var downloaded int64

	u.mu.RLock()
	files := u.cfg.Files
	u.mu.RUnlock()

	for _, file := range manifest.Files {
		u.reportProgress(iot.UpdateProgress{
			State:           iot.StateDownloading,
			CurrentFile:     file.Name,
			BytesDownloaded: downloaded,
			TotalBytes:      totalSize,
			Percentage:      percent(downloaded, totalSize),
		})

		data, ok := files[file.Name]
		if !ok {
			err := iot.ErrDownloadFailed(file.Name, nil)
			u.reportProgress(iot.UpdateProgress{State: iot.StateFailed, Error: err.Error()})
			return nil, err
		}

		sum := sha256.Sum256(data)
		got := hex.EncodeToString(sum[:])
		if file.SHA256 != "" && got != file.SHA256 {
			err := iot.ErrChecksumMismatch(file.Name, file.SHA256, got)
			u.reportProgress(iot.UpdateProgress{State: iot.StateFailed, Error: err.Error()})
			return nil, err
		}

		out[file.Name] = append([]byte(nil), data...)
		downloaded += file.Size
	}

	u.reportProgress(iot.UpdateProgress{
		State:           iot.StateVerifying,
		BytesDownloaded: totalSize,
		TotalBytes:      totalSize,
		Percentage:      100,
	})
	return out, nil
}

// ApplyUpdate marks the update complete (or delegates to ApplyFn).
func (u *Updater) ApplyUpdate(ctx context.Context, files map[string][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	u.reportProgress(iot.UpdateProgress{State: iot.StateInstalling})

	u.mu.RLock()
	fn := u.cfg.ApplyFn
	u.mu.RUnlock()

	if fn != nil {
		if err := fn(ctx, files); err != nil {
			u.reportProgress(iot.UpdateProgress{State: iot.StateFailed, Error: err.Error()})
			return iot.ErrUpdateFailed(err)
		}
	}

	u.reportProgress(iot.UpdateProgress{State: iot.StateComplete})
	return nil
}

// CheckAndApply checks for updates and applies when available.
func (u *Updater) CheckAndApply(ctx context.Context, deviceID, currentVersion string) error {
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
func (u *Updater) GetState() iot.UpdateState {
	if v, ok := u.state.Load().(iot.UpdateState); ok {
		return v
	}
	return iot.StateIdle
}

func percent(downloaded, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(downloaded) / float64(total) * 100
}

func cloneManifest(m *iot.UpdateManifest) *iot.UpdateManifest {
	if m == nil {
		return nil
	}
	cp := *m
	if m.Files != nil {
		cp.Files = append([]iot.UpdateFile(nil), m.Files...)
	}
	if m.Metadata != nil {
		cp.Metadata = make(map[string]string, len(m.Metadata))
		for k, v := range m.Metadata {
			cp.Metadata[k] = v
		}
	}
	return &cp
}

// FileSHA256 returns the hex-encoded SHA-256 of data (test helper).
func FileSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
