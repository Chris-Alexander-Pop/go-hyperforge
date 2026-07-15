// Package google provides a thin Google Cloud Speech-to-Text / Text-to-Speech HTTP client.
//
// Uses injectable *http.Client and REST endpoints (no Google Cloud SDK dependency).
// Point Endpoint / TTSEndpoint at Google APIs or httptest doubles.
package google

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Config configures the Google speech HTTP client.
type Config struct {
	// APIKey is sent as ?key= for API-key auth (optional if Authorization is set).
	APIKey string

	// Authorization is an optional Bearer token (OAuth access token).
	Authorization string

	// STTEndpoint defaults to the Speech-to-Text v1 recognize URL.
	STTEndpoint string

	// TTSEndpoint defaults to the Text-to-Speech v1 synthesize URL.
	TTSEndpoint string

	// LanguageCode defaults to en-US.
	LanguageCode string

	// VoiceName is the TTS voice (e.g. en-US-Neural2-A).
	VoiceName string

	// HTTPClient overrides the default 60s client.
	HTTPClient *http.Client
}

// Client implements speech.SpeechClient over Google Cloud REST APIs.
type Client struct {
	cfg  Config
	http *http.Client
}

// New creates a Google speech HTTP client.
func New(cfg Config) *Client {
	if cfg.STTEndpoint == "" {
		cfg.STTEndpoint = "https://speech.googleapis.com/v1/speech:recognize"
	}
	if cfg.TTSEndpoint == "" {
		cfg.TTSEndpoint = "https://texttospeech.googleapis.com/v1/text:synthesize"
	}
	if cfg.LanguageCode == "" {
		cfg.LanguageCode = "en-US"
	}
	if cfg.VoiceName == "" {
		cfg.VoiceName = "en-US-Neural2-A"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{cfg: cfg, http: hc}
}

func (c *Client) authURL(base string) string {
	if c.cfg.APIKey == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "key=" + c.cfg.APIKey
}

func (c *Client) doJSON(ctx context.Context, url string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, pkgerrors.Internal("marshal request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL(url), bytes.NewReader(body))
	if err != nil {
		return nil, pkgerrors.Internal("create request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Authorization != "" {
		req.Header.Set("Authorization", c.cfg.Authorization)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, pkgerrors.Unavailable("google speech request failed", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.Internal("read response", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, speech.MapHTTPStatus(resp.StatusCode, string(raw))
	}
	return raw, nil
}

// SpeechToText calls speech:recognize.
func (c *Client) SpeechToText(ctx context.Context, audio []byte) (string, error) {
	if len(audio) == 0 {
		return "", speech.ErrEmptyAudio
	}
	payload := map[string]any{
		"config": map[string]any{
			"languageCode": c.cfg.LanguageCode,
			"encoding":     "LINEAR16",
		},
		"audio": map[string]any{
			"content": base64.StdEncoding.EncodeToString(audio),
		},
	}
	raw, err := c.doJSON(ctx, c.cfg.STTEndpoint, payload)
	if err != nil {
		return "", err
	}
	var out struct {
		Results []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
			} `json:"alternatives"`
		} `json:"results"`
		Text string `json:"text"` // test-friendly shorthand
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", pkgerrors.Internal("parse STT response", err)
	}
	if out.Text != "" {
		return out.Text, nil
	}
	if len(out.Results) > 0 && len(out.Results[0].Alternatives) > 0 {
		return out.Results[0].Alternatives[0].Transcript, nil
	}
	return "", nil
}

// TextToSpeech calls text:synthesize.
func (c *Client) TextToSpeech(ctx context.Context, text string, format speech.AudioFormat) ([]byte, error) {
	if text == "" {
		return nil, speech.ErrEmptyText
	}
	encoding := "MP3"
	switch format {
	case speech.FormatWAV, speech.FormatFLAC:
		encoding = "LINEAR16"
	case speech.FormatOGG:
		encoding = "OGG_OPUS"
	case speech.FormatMP3, "":
		encoding = "MP3"
	}
	payload := map[string]any{
		"input": map[string]string{"text": text},
		"voice": map[string]string{
			"languageCode": c.cfg.LanguageCode,
			"name":         c.cfg.VoiceName,
		},
		"audioConfig": map[string]string{"audioEncoding": encoding},
	}
	raw, err := c.doJSON(ctx, c.cfg.TTSEndpoint, payload)
	if err != nil {
		return nil, err
	}
	var out struct {
		AudioContent string `json:"audioContent"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, pkgerrors.Internal("parse TTS response", err)
	}
	if out.AudioContent == "" {
		return raw, nil // allow raw audio in tests
	}
	decoded, err := base64.StdEncoding.DecodeString(out.AudioContent)
	if err != nil {
		return nil, pkgerrors.Internal("decode audioContent", err)
	}
	return decoded, nil
}

var _ speech.SpeechClient = (*Client)(nil)
