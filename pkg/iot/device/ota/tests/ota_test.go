package ota_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/iot"
	"github.com/chris-alexander-pop/system-design-library/pkg/iot/device/ota"
)

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestHTTPUpdater_SemverAndChecksum(t *testing.T) {
	fw := []byte("ota-firmware-body")
	sum := sha256Hex(fw)
	var downloads atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_ = json.NewEncoder(w).Encode(ota.UpdateManifest{
				Version:     "1.10.0",
				Description: "semver fix",
				ReleaseDate: time.Now().UTC(),
				Files: []ota.UpdateFile{{
					Name:   "app.bin",
					URL:    "http://" + r.Host + "/files/app.bin",
					Size:   int64(len(fw)),
					SHA256: sum,
				}},
			})
		case "/files/app.bin":
			downloads.Add(1)
			_, _ = w.Write(fw)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	u := ota.New(ota.Config{
		StorageURL:   srv.URL,
		ManifestPath: "/manifest.json",
		MaxRetries:   2,
		Timeout:      5 * time.Second,
	})

	manifest, available, err := u.CheckForUpdate(context.Background(), "1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if !available {
		t.Fatal("1.10.0 must be newer than 1.9.0 (semver, not string compare)")
	}

	_, available, err = u.CheckForUpdate(context.Background(), "1.10.0")
	if err != nil {
		t.Fatal(err)
	}
	if available {
		t.Fatal("same version should not be available")
	}

	files, err := u.DownloadUpdate(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if string(files["app.bin"]) != string(fw) {
		t.Fatal("bad firmware")
	}
	if downloads.Load() < 1 {
		t.Fatal("expected download")
	}

	if err := u.ApplyUpdate(context.Background(), files); err != nil {
		t.Fatal(err)
	}
	if u.GetState() != ota.StateComplete {
		t.Fatalf("state=%s", u.GetState())
	}
}

func TestHTTPUpdater_ChecksumMismatch(t *testing.T) {
	fw := []byte("body")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_ = json.NewEncoder(w).Encode(ota.UpdateManifest{
				Version: "2.0.0",
				Files: []ota.UpdateFile{{
					Name:   "app.bin",
					URL:    "http://" + r.Host + "/app.bin",
					Size:   int64(len(fw)),
					SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
				}},
			})
		case "/app.bin":
			_, _ = w.Write(fw)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	u := ota.New(ota.Config{
		StorageURL:   srv.URL,
		ManifestPath: "/manifest.json",
		MaxRetries:   2,
	})

	manifest, available, err := u.CheckForUpdate(context.Background(), "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if !available {
		t.Fatal("expected available")
	}

	_, err = u.DownloadUpdate(context.Background(), manifest)
	if err == nil {
		t.Fatal("expected checksum failure")
	}
	if !pkgerrors.IsCode(err, iot.CodeDownloadFailed) && !pkgerrors.IsCode(err, iot.CodeChecksumMismatch) {
		t.Fatalf("unexpected err: %v code=%s", err, pkgerrors.Code(err))
	}
}

func TestHTTPUpdater_CheckAndApply_NoUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ota.UpdateManifest{Version: "1.0.0"})
	}))
	defer srv.Close()

	u := ota.New(ota.Config{StorageURL: srv.URL, ManifestPath: "/"})
	if err := u.CheckAndApply(context.Background(), "d1", "1.0.0"); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPUpdater_ManifestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	u := ota.New(ota.Config{StorageURL: srv.URL, ManifestPath: "/missing"})
	_, _, err := u.CheckForUpdate(context.Background(), "1.0.0")
	if !pkgerrors.IsCode(err, iot.CodeManifestNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestHTTPUpdater_ImplementsInterface(t *testing.T) {
	var _ iot.Updater = ota.New(ota.Config{})
}
