package google_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech/adapters/google"
)

func TestGoogleSTTAndTTS(t *testing.T) {
	stt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"alternatives": []map[string]string{{"transcript": "hello google"}}},
			},
		})
	}))
	defer stt.Close()

	tts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"audioContent": base64.StdEncoding.EncodeToString([]byte("wave")),
		})
	}))
	defer tts.Close()

	c := google.New(google.Config{
		APIKey:      "test-key",
		STTEndpoint: stt.URL,
		TTSEndpoint: tts.URL,
	})

	text, err := c.SpeechToText(context.Background(), []byte{1})
	if err != nil || text != "hello google" {
		t.Fatalf("STT: %q %v", text, err)
	}
	audio, err := c.TextToSpeech(context.Background(), "hi", speech.FormatMP3)
	if err != nil || string(audio) != "wave" {
		t.Fatalf("TTS: %q %v", audio, err)
	}
}
