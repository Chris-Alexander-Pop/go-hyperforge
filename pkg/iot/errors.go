package iot

import (
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Error codes for IoT operations.
const (
	CodeConnectionFailed = "IOT_CONN_FAILED"
	CodePublishFailed    = "IOT_PUBLISH_FAILED"
	CodeSubscribeFailed  = "IOT_SUBSCRIBE_FAILED"
	CodeTimeout          = "IOT_TIMEOUT"
	CodeNotConnected     = "IOT_NOT_CONNECTED"
	CodeInvalidConfig    = "IOT_INVALID_CONFIG"
	CodeManifestNotFound = "IOT_MANIFEST_NOT_FOUND"
	CodeDownloadFailed   = "IOT_DOWNLOAD_FAILED"
	CodeChecksumMismatch = "IOT_CHECKSUM_MISMATCH"
	CodeInvalidVersion   = "IOT_INVALID_VERSION"
	CodeUpdateFailed     = "IOT_UPDATE_FAILED"
)

// ErrConnectionFailed creates an error for MQTT broker connection failures.
func ErrConnectionFailed(err error) *errors.AppError {
	return errors.New(CodeConnectionFailed, "failed to connect to MQTT broker", err)
}

// ErrPublishFailed creates an error for MQTT publish failures.
func ErrPublishFailed(err error) *errors.AppError {
	return errors.New(CodePublishFailed, "failed to publish MQTT message", err)
}

// ErrSubscribeFailed creates an error for MQTT subscribe failures.
func ErrSubscribeFailed(err error) *errors.AppError {
	return errors.New(CodeSubscribeFailed, "failed to subscribe to MQTT topic", err)
}

// ErrTimeout creates an error for timed-out MQTT operations.
func ErrTimeout(operation string, err error) *errors.AppError {
	return errors.New(CodeTimeout, "MQTT operation timed out: "+operation, err)
}

// ErrNotConnected creates an error when the client is not connected.
func ErrNotConnected() *errors.AppError {
	return errors.New(CodeNotConnected, "MQTT client is not connected", nil)
}

// ErrInvalidConfig creates an error for invalid configuration.
func ErrInvalidConfig(msg string, err error) *errors.AppError {
	return errors.New(CodeInvalidConfig, "invalid IoT configuration: "+msg, err)
}

// ErrManifestNotFound creates an error when the OTA manifest cannot be fetched.
func ErrManifestNotFound(err error) *errors.AppError {
	return errors.New(CodeManifestNotFound, "OTA manifest not found", err)
}

// ErrDownloadFailed creates an error for OTA download failures.
func ErrDownloadFailed(name string, err error) *errors.AppError {
	return errors.New(CodeDownloadFailed, fmt.Sprintf("failed to download update file %q", name), err)
}

// ErrChecksumMismatch creates an error when a downloaded file fails SHA-256 verification.
func ErrChecksumMismatch(name, expected, got string) *errors.AppError {
	return errors.New(CodeChecksumMismatch,
		fmt.Sprintf("checksum mismatch for %q: expected %s, got %s", name, expected, got), nil)
}

// ErrInvalidVersion creates an error for unparseable semantic versions.
func ErrInvalidVersion(version string, err error) *errors.AppError {
	return errors.New(CodeInvalidVersion, fmt.Sprintf("invalid semantic version %q", version), err)
}

// ErrUpdateFailed creates an error for OTA apply failures.
func ErrUpdateFailed(err error) *errors.AppError {
	return errors.New(CodeUpdateFailed, "OTA update failed", err)
}
