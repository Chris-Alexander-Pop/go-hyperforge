// Package ota provides Over-the-Air update functionality for IoT devices.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/iot/device/ota"
//
//	updater := ota.New(ota.Config{StorageURL: "s3://updates-bucket"})
//	err := updater.CheckAndApply(ctx, "device-123", "1.0.0")
package ota

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// UpdateState represents the current update state.
type UpdateState string

const (
	StateIdle        UpdateState = "idle"
	StateChecking    UpdateState = "checking"
	StateDownloading UpdateState = "downloading"
	StateVerifying   UpdateState = "verifying"
	StateInstalling  UpdateState = "installing"
	StateRebooting   UpdateState = "rebooting"
	StateFailed      UpdateState = "failed"
	StateComplete    UpdateState = "complete"
)

// Config holds OTA configuration.
type Config struct {
	// StorageURL is the base URL for update files
	StorageURL string

	// ManifestPath is the path to the update manifest
	ManifestPath string

	// DownloadDir is where updates are downloaded
	DownloadDir string

	// Timeout for HTTP requests
	Timeout time.Duration

	// MaxRetries for failed downloads
	MaxRetries int
}

// UpdateManifest describes an available update.
type UpdateManifest struct {
	Version     string            `json:"version"`
	Description string            `json:"description"`
	ReleaseDate time.Time         `json:"release_date"`
	Files       []UpdateFile      `json:"files"`
	MinVersion  string            `json:"min_version,omitempty"`
	MaxVersion  string            `json:"max_version,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UpdateFile describes a single update file.
type UpdateFile struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Required bool   `json:"required"`
}

// UpdateProgress reports download/install progress.
type UpdateProgress struct {
	State           UpdateState `json:"state"`
	CurrentFile     string      `json:"current_file,omitempty"`
	BytesDownloaded int64       `json:"bytes_downloaded"`
	TotalBytes      int64       `json:"total_bytes"`
	Percentage      float64     `json:"percentage"`
	Error           string      `json:"error,omitempty"`
}

// ProgressCallback is called with update progress.
type ProgressCallback func(progress UpdateProgress)

// Updater manages OTA updates.
type Updater struct {
	config     Config
	httpClient *http.Client
	progress   ProgressCallback
	state      UpdateState
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

	return &Updater{
		config:     cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		state:      StateIdle,
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

// CheckForUpdate checks if an update is available.
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
		return nil, false, pkgerrors.NotFound("manifest not found", nil)
	}

	var manifest UpdateManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, false, pkgerrors.Internal("failed to parse manifest", err)
	}

	u.reportProgress(UpdateProgress{State: StateIdle})

	// Simple version comparison (assumes semver-like format)
	if manifest.Version > currentVersion {
		return &manifest, true, nil
	}

	return &manifest, false, nil
}

// DownloadUpdate downloads update files.
func (u *Updater) DownloadUpdate(ctx context.Context, manifest *UpdateManifest) (map[string][]byte, error) {
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
			Percentage:      float64(downloaded) / float64(totalSize) * 100,
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
	var lastErr error

	for attempt := 0; attempt < u.config.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", file.URL, http.NoBody)
		if err != nil {
			return nil, pkgerrors.Internal("failed to create request", err)
		}

		resp, err := u.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		// Verify checksum
		hash := sha256.Sum256(data)
		checksum := hex.EncodeToString(hash[:])
		if checksum != file.SHA256 {
			lastErr = fmt.Errorf("checksum mismatch: expected %s, got %s", file.SHA256, checksum)
			continue
		}

		return data, nil
	}

	return nil, pkgerrors.Internal("failed to download file after retries", lastErr)
}

// ApplyUpdate applies downloaded updates (platform-specific implementation needed).
func (u *Updater) ApplyUpdate(ctx context.Context, files map[string][]byte) error {
	u.reportProgress(UpdateProgress{State: StateInstalling})

	// Platform-specific update logic would go here
	// This is a stub that simulates the process

	u.reportProgress(UpdateProgress{State: StateComplete})
	return nil
}

// CheckAndApply checks for updates and applies if available.
func (u *Updater) CheckAndApply(ctx context.Context, deviceID, currentVersion string) error {
	manifest, available, err := u.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		return err
	}

	if !available {
		return nil // No update available
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
