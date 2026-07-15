package memory_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/adapters/memory"
)

func TestClient_PublishSubscribe(t *testing.T) {
	ctx := context.Background()
	c := memory.NewClient()
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect()

	var got *iot.Message
	var mu sync.Mutex
	err := c.Subscribe(ctx, "sensors/temp", func(msg *iot.Message) {
		mu.Lock()
		got = msg
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{"c":22.5}`)
	if err := c.Publish(ctx, "sensors/temp", payload); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if got == nil {
		t.Fatal("expected message delivery")
	}
	if got.Topic != "sensors/temp" {
		t.Fatalf("topic = %q", got.Topic)
	}
	if string(got.Payload) != string(payload) {
		t.Fatalf("payload = %q", got.Payload)
	}
	if got.MessageID == 0 {
		t.Fatal("expected non-zero message id")
	}
}

func TestClient_WildcardSubscribe(t *testing.T) {
	ctx := context.Background()
	c := memory.NewClient()
	_ = c.Connect(ctx)

	var hits int
	_ = c.Subscribe(ctx, "sensors/+/temp", func(msg *iot.Message) { hits++ })
	_ = c.Publish(ctx, "sensors/a/temp", []byte("1"))
	_ = c.Publish(ctx, "sensors/b/humidity", []byte("2"))
	if hits != 1 {
		t.Fatalf("hits = %d, want 1", hits)
	}

	hits = 0
	_ = c.Subscribe(ctx, "devices/#", func(msg *iot.Message) { hits++ })
	_ = c.Publish(ctx, "devices/1/status", []byte("ok"))
	if hits != 1 {
		t.Fatalf("multilevel hits = %d, want 1", hits)
	}
}

func TestClient_NotConnected(t *testing.T) {
	ctx := context.Background()
	c := memory.NewClient()
	err := c.Publish(ctx, "t", []byte("x"))
	if err == nil {
		t.Fatal("expected not connected error")
	}
	if !pkgerrors.IsCode(err, iot.CodeNotConnected) {
		t.Fatalf("code = %s", pkgerrors.Code(err))
	}
}

func TestClient_Unsubscribe(t *testing.T) {
	ctx := context.Background()
	c := memory.NewClient()
	_ = c.Connect(ctx)

	var hits int
	_ = c.Subscribe(ctx, "t", func(msg *iot.Message) { hits++ })
	_ = c.Unsubscribe(ctx, "t")
	_ = c.Publish(ctx, "t", []byte("x"))
	if hits != 0 {
		t.Fatalf("hits after unsubscribe = %d", hits)
	}
}

func TestClient_CanceledContext(t *testing.T) {
	c := memory.NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Connect(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestClient_Instrumented(t *testing.T) {
	ctx := context.Background()
	raw := memory.NewClient()
	c := iot.NewInstrumentedClient(raw)
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if !c.IsConnected() {
		t.Fatal("expected connected")
	}
	_ = c.Subscribe(ctx, "t", func(msg *iot.Message) {})
	if err := c.Publish(ctx, "t", []byte("hi")); err != nil {
		t.Fatal(err)
	}
	if err := c.PublishWithOptions(ctx, "t", []byte("hi"), iot.QoSExactlyOnce, true); err != nil {
		t.Fatal(err)
	}
	_ = c.Unsubscribe(ctx, "t")
	c.Disconnect()
}

func TestUpdater_ChecksumAndVersion(t *testing.T) {
	ctx := context.Background()
	fw := []byte("firmware-v2-bytes")
	sum := memory.FileSHA256(fw)

	manifest := &iot.UpdateManifest{
		Version:     "1.10.0",
		Description: "bump",
		ReleaseDate: time.Now().UTC(),
		Files: []iot.UpdateFile{{
			Name:   "app.bin",
			URL:    "memory://app.bin",
			Size:   int64(len(fw)),
			SHA256: sum,
		}},
	}

	u := memory.NewUpdater(memory.UpdaterConfig{
		Manifest: manifest,
		Files:    map[string][]byte{"app.bin": fw},
	})

	// String compare would wrongly treat "1.9.0" > "1.10.0"; semver must detect newer.
	m, available, err := u.CheckForUpdate(ctx, "1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if !available {
		t.Fatal("expected update available for 1.10.0 > 1.9.0")
	}
	if m.Version != "1.10.0" {
		t.Fatalf("version = %s", m.Version)
	}

	_, available, err = u.CheckForUpdate(ctx, "1.10.0")
	if err != nil {
		t.Fatal(err)
	}
	if available {
		t.Fatal("expected no update when current == manifest")
	}

	_, available, err = u.CheckForUpdate(ctx, "2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if available {
		t.Fatal("expected no update when current is newer")
	}

	files, err := u.DownloadUpdate(ctx, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if string(files["app.bin"]) != string(fw) {
		t.Fatal("unexpected payload")
	}
	if u.GetState() != iot.StateVerifying {
		t.Fatalf("state = %s", u.GetState())
	}
}

func TestUpdater_ChecksumMismatch(t *testing.T) {
	ctx := context.Background()
	fw := []byte("good")
	manifest := &iot.UpdateManifest{
		Version: "2.0.0",
		Files: []iot.UpdateFile{{
			Name:   "app.bin",
			Size:   int64(len(fw)),
			SHA256: "deadbeef",
		}},
	}
	u := memory.NewUpdater(memory.UpdaterConfig{
		Manifest: manifest,
		Files:    map[string][]byte{"app.bin": fw},
	})

	_, err := u.DownloadUpdate(ctx, manifest)
	if err == nil {
		t.Fatal("expected checksum mismatch")
	}
	if !pkgerrors.IsCode(err, iot.CodeChecksumMismatch) {
		t.Fatalf("code = %s err=%v", pkgerrors.Code(err), err)
	}
	if u.GetState() != iot.StateFailed {
		t.Fatalf("state = %s", u.GetState())
	}
}

func TestUpdater_MissingFile(t *testing.T) {
	ctx := context.Background()
	manifest := &iot.UpdateManifest{
		Version: "2.0.0",
		Files:   []iot.UpdateFile{{Name: "missing.bin", Size: 1, SHA256: "abc"}},
	}
	u := memory.NewUpdater(memory.UpdaterConfig{Manifest: manifest})
	_, err := u.DownloadUpdate(ctx, manifest)
	if err == nil {
		t.Fatal("expected download failure")
	}
	if !pkgerrors.IsCode(err, iot.CodeDownloadFailed) {
		t.Fatalf("code = %s", pkgerrors.Code(err))
	}
}

func TestUpdater_CheckAndApply(t *testing.T) {
	ctx := context.Background()
	fw := []byte("payload")
	sum := memory.FileSHA256(fw)
	applied := false

	u := memory.NewUpdater(memory.UpdaterConfig{
		Manifest: &iot.UpdateManifest{
			Version: "v3.0.0",
			Files:   []iot.UpdateFile{{Name: "f.bin", Size: int64(len(fw)), SHA256: sum}},
		},
		Files: map[string][]byte{"f.bin": fw},
		ApplyFn: func(ctx context.Context, files map[string][]byte) error {
			applied = true
			if string(files["f.bin"]) != string(fw) {
				return errors.New("bad file")
			}
			return nil
		},
	})

	var last iot.UpdateProgress
	u.SetProgressCallback(func(p iot.UpdateProgress) { last = p })

	if err := u.CheckAndApply(ctx, "dev-1", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("expected ApplyFn")
	}
	if u.GetState() != iot.StateComplete {
		t.Fatalf("state = %s", u.GetState())
	}
	if last.State != iot.StateComplete {
		t.Fatalf("last progress = %s", last.State)
	}

	applied = false
	if err := u.CheckAndApply(ctx, "dev-1", "3.0.0"); err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("should not apply when up to date")
	}
}

func TestUpdater_ManifestNotFound(t *testing.T) {
	u := memory.NewUpdater(memory.UpdaterConfig{})
	_, _, err := u.CheckForUpdate(context.Background(), "1.0.0")
	if !pkgerrors.IsCode(err, iot.CodeManifestNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdater_InvalidVersion(t *testing.T) {
	u := memory.NewUpdater(memory.UpdaterConfig{
		Manifest: &iot.UpdateManifest{Version: "not-a-version"},
	})
	_, _, err := u.CheckForUpdate(context.Background(), "1.0.0")
	if !pkgerrors.IsCode(err, iot.CodeInvalidVersion) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdater_Instrumented(t *testing.T) {
	fw := []byte("x")
	sum := memory.FileSHA256(fw)
	raw := memory.NewUpdater(memory.UpdaterConfig{
		Manifest: &iot.UpdateManifest{
			Version: "1.1.0",
			Files:   []iot.UpdateFile{{Name: "a", Size: 1, SHA256: sum}},
		},
		Files: map[string][]byte{"a": fw},
	})
	u := iot.NewInstrumentedUpdater(raw)
	u.SetProgressCallback(func(p iot.UpdateProgress) {})
	if err := u.CheckAndApply(context.Background(), "d", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if u.GetState() != iot.StateComplete {
		t.Fatalf("state = %s", u.GetState())
	}
}
