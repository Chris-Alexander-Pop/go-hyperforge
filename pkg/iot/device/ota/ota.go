package ota

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/iot"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

// Re-export shared OTA types from the root iot package.
type (
	UpdateState      = iot.UpdateState
	UpdateManifest   = iot.UpdateManifest
	UpdateFile       = iot.UpdateFile
	UpdateProgress   = iot.UpdateProgress
	ProgressCallback = iot.ProgressCallback
)

// State constants (aliases of iot states).
const (
	StateIdle        = iot.StateIdle
	StateChecking    = iot.StateChecking
	StateDownloading = iot.StateDownloading
	StateVerifying   = iot.StateVerifying
	StateInstalling  = iot.StateInstalling
	StateRebooting   = iot.StateRebooting
	StateFailed      = iot.StateFailed
	StateComplete    = iot.StateComplete
)

// Ensure Updater implements iot.Updater.
var _ iot.Updater = (*Updater)(nil)

// Config holds OTA configuration.
type Config struct {
	// StorageURL is the base URL for update files
	StorageURL string

	// ManifestPath is the path to the update manifest
	ManifestPath string

	// DownloadDir is where updates are downloaded (reserved for future file-backed installs)
	DownloadDir string

	// Timeout for HTTP requests
	Timeout time.Duration

	// MaxRetries for failed downloads (passed to pkg/resilience)
	MaxRetries int
}

// Updater manages OTA updates over HTTP.
type Updater struct {
	config     Config
	httpClient *http.Client
	progress   ProgressCallback
	state      UpdateState
	retrier    resilience.Retrier
}

// New creates a new OTA updater.
func New(cfg Config) *Updater {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.ManifestPath == "" {
		cfg.ManifestPath = "/manifest.json"
	}

	retryCfg := resilience.DefaultRetryConfig()
	retryCfg.MaxAttempts = cfg.MaxRetries
	retryCfg.InitialBackoff = 50 * time.Millisecond
	retryCfg.MaxBackoff = 2 * time.Second

	return &Updater{
		config:     cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		state:      StateIdle,
		retrier:    resilience.NewRetrier(retryCfg),
	}
}

// SetProgressCallback sets the progress callback.
func (u *Updater) SetProgressCallback(cb ProgressCallback) {
	u.progress = cb
}

func (u *Updater) reportProgress(p UpdateProgress) {
	u.state = p.State
	if u.progress != nil {
		u.progress(p)
	}
}

// CheckForUpdate checks if an update is available using semantic version comparison.
func (u *Updater) CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateManifest, bool, error) {
	u.reportProgress(UpdateProgress{State: StateChecking})

	manifestURL := u.config.StorageURL + u.config.ManifestPath
	req, err := http.NewRequestWithContext(ctx, "GET", manifestURL, http.NoBody)
	if err != nil {
		return nil, false, pkgerrors.Internal("failed to create request", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, false, pkgerrors.Internal("failed to fetch manifest", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, iot.ErrManifestNotFound(nil)
	}

	var manifest UpdateManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
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

// DownloadUpdate downloads update files with checksum verification and retries.
func (u *Updater) DownloadUpdate(ctx context.Context, manifest *UpdateManifest) (map[string][]byte, error) {
	if manifest == nil {
		return nil, iot.ErrInvalidConfig("manifest is required", nil)
	}
	u.reportProgress(UpdateProgress{State: StateDownloading})

	var totalSize int64
	for _, f := range manifest.Files {
		totalSize += f.Size
	}

	var downloaded int64
	files := make(map[string][]byte)

	for _, file := range manifest.Files {
		u.reportProgress(UpdateProgress{
			State:           StateDownloading,
			CurrentFile:     file.Name,
			BytesDownloaded: downloaded,
			TotalBytes:      totalSize,
			Percentage:      percent(downloaded, totalSize),
		})

		data, err := u.downloadFile(ctx, file)
		if err != nil {
			u.reportProgress(UpdateProgress{
				State: StateFailed,
				Error: err.Error(),
			})
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

func (u *Updater) downloadFile(ctx context.Context, file UpdateFile) ([]byte, error) {
	var data []byte

	err := u.retrier.Execute(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", file.URL, http.NoBody)
		if err != nil {
			return pkgerrors.Internal("failed to create request", err)
		}

		resp, err := u.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		hash := sha256.Sum256(body)
		checksum := hex.EncodeToString(hash[:])
		if file.SHA256 != "" && checksum != file.SHA256 {
			return iot.ErrChecksumMismatch(file.Name, file.SHA256, checksum)
		}

		data = body
		return nil
	})
	if err != nil {
		return nil, iot.ErrDownloadFailed(file.Name, err)
	}
	return data, nil
}

// ApplyUpdate applies downloaded updates (stub — platform-specific logic required).
func (u *Updater) ApplyUpdate(ctx context.Context, files map[string][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	u.reportProgress(UpdateProgress{State: StateInstalling})
	_ = files
	u.reportProgress(UpdateProgress{State: StateComplete})
	return nil
}

// CheckAndApply checks for updates and applies if available.
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
func (u *Updater) GetState() UpdateState {
	return u.state
}

func percent(downloaded, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(downloaded) / float64(total) * 100
}
