package google_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech/adapters/google"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func TestGoogleHTTPStatusMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := google.New(google.Config{
		STTEndpoint: srv.URL,
		TTSEndpoint: srv.URL,
	})
	_, err := c.TextToSpeech(context.Background(), "hi", speech.FormatMP3)
	if !errors.IsCode(err, errors.CodeUnavailable) {
		t.Fatalf("expected UNAVAILABLE, got %v", err)
	}
	_, err = c.SpeechToText(context.Background(), []byte{1})
	if !errors.IsCode(err, errors.CodeUnavailable) {
		t.Fatalf("expected UNAVAILABLE, got %v", err)
	}
}
