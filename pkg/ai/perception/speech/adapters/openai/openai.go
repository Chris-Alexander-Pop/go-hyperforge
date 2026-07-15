// Package openai provides an OpenAI Whisper adapter for speech transcription.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Service implements speech.SpeechClient using OpenAI.
type Service struct {
	apiKey     string
	httpClient *http.Client
}

// New creates a new OpenAI speech service.
func New(apiKey string) *Service {
	return &Service{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *Service) SpeechToText(ctx context.Context, audio []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("model", "whisper-1")

	// Default filename since we receive raw bytes
	part, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", pkgerrors.Internal("failed to create form file", err)
	}
	_, err = part.Write(audio)
	if err != nil {
		return "", pkgerrors.Internal("failed to write audio", err)
	}

	err = writer.Close()
	if err != nil {
		return "", pkgerrors.Internal("failed to close writer", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", body)
	if err != nil {
		return "", pkgerrors.Internal("failed to create request", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", pkgerrors.Internal("API request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", pkgerrors.Internal("API error: "+string(respBody), nil)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", pkgerrors.Internal("failed to parse response", err)
	}

	return result.Text, nil
}

func (s *Service) TextToSpeech(ctx context.Context, text string, format speech.AudioFormat) ([]byte, error) {
	responseFormat := "mp3"
	if format != "" {
		responseFormat = string(format)
	}

	reqBody, _ := json.Marshal(map[string]string{
		"model":           "tts-1",
		"input":           text,
		"voice":           "alloy",
		"response_format": responseFormat,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/speech", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("API request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, pkgerrors.Internal("API error: "+string(respBody), nil)
	}

	return io.ReadAll(resp.Body)
}

var _ speech.SpeechClient = (*Service)(nil)
