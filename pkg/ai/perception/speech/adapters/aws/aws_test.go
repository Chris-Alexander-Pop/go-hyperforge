package aws_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech/adapters/aws"
)

type fakePolly struct{}

func (fakePolly) SynthesizeSpeech(ctx context.Context, text string, format speech.AudioFormat) ([]byte, error) {
	return []byte("polly:" + text + ":" + string(format)), nil
}

type fakeTranscribe struct{}

func (fakeTranscribe) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "transcribed", nil
}

func TestAWSInjectedSDK(t *testing.T) {
	c := aws.New(aws.Config{}, aws.WithPolly(fakePolly{}), aws.WithTranscribe(fakeTranscribe{}))
	text, err := c.SpeechToText(context.Background(), []byte("audio"))
	if err != nil || text != "transcribed" {
		t.Fatalf("STT: %q %v", text, err)
	}
	audio, err := c.TextToSpeech(context.Background(), "hi", speech.FormatMP3)
	if err != nil || string(audio) != "polly:hi:mp3" {
		t.Fatalf("TTS: %q %v", audio, err)
	}
}

func TestAWSHTTPEndpoints(t *testing.T) {
	pollySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var in map[string]string
		_ = json.Unmarshal(body, &in)
		_, _ = w.Write([]byte("audio-" + in["text"]))
	}))
	defer pollySrv.Close()

	txSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"text": "hello"})
	}))
	defer txSrv.Close()

	c := aws.New(aws.Config{
		PollyEndpoint:      pollySrv.URL,
		TranscribeEndpoint: txSrv.URL,
	})
	audio, err := c.TextToSpeech(context.Background(), "world", speech.FormatWAV)
	if err != nil || string(audio) != "audio-world" {
		t.Fatalf("TTS HTTP: %q %v", audio, err)
	}
	text, err := c.SpeechToText(context.Background(), []byte{1, 2})
	if err != nil || text != "hello" {
		t.Fatalf("STT HTTP: %q %v", text, err)
	}
}
