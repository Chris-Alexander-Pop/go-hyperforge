package aws_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech/adapters/aws"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func TestAWSHTTPStatusMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := aws.New(aws.Config{
		PollyEndpoint:      srv.URL,
		TranscribeEndpoint: srv.URL,
	})
	_, err := c.TextToSpeech(context.Background(), "hi", speech.FormatMP3)
	if !errors.IsCode(err, errors.CodeResourceExhausted) {
		t.Fatalf("expected RESOURCE_EXHAUSTED, got %v", err)
	}
	_, err = c.SpeechToText(context.Background(), []byte{1})
	if !errors.IsCode(err, errors.CodeResourceExhausted) {
		t.Fatalf("expected RESOURCE_EXHAUSTED, got %v", err)
	}
}

func TestAWSEmptyInputs(t *testing.T) {
	c := aws.New(aws.Config{}, aws.WithPolly(fakePolly{}), aws.WithTranscribe(fakeTranscribe{}))
	if _, err := c.SpeechToText(context.Background(), nil); err != speech.ErrEmptyAudio {
		t.Fatalf("empty audio: %v", err)
	}
	if _, err := c.TextToSpeech(context.Background(), "", speech.FormatMP3); err != speech.ErrEmptyText {
		t.Fatalf("empty text: %v", err)
	}
}
