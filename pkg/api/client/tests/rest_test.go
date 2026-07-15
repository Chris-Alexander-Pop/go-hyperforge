package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/client/rest"
)

func TestNewRESTClient(t *testing.T) {
	// No BaseURL in Config, it's per-request or standard
	cfg := rest.Config{
		Timeout: time.Second,
	}

	client, err := rest.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client, got nil")
	}
}
