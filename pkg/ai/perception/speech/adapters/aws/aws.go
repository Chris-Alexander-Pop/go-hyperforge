// Package aws provides an injectable AWS Polly (TTS) + Transcribe (STT) speech adapter.
//
// Real AWS SDK clients can be injected via PollyAPI / TranscribeAPI; tests use fakes.
// This package does not import the AWS SDK so dependents stay light.
package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// PollyAPI synthesizes speech (injectable; wrap aws-sdk-go-v2 Polly in production).
type PollyAPI interface {
	SynthesizeSpeech(ctx context.Context, text string, format speech.AudioFormat) ([]byte, error)
}

// TranscribeAPI converts audio to text (injectable).
type TranscribeAPI interface {
	Transcribe(ctx context.Context, audio []byte) (string, error)
}

// Config configures the AWS speech client.
type Config struct {
	// Region is informational / used by HTTP helpers.
	Region string

	// VoiceID is the Polly voice (default Joanna).
	VoiceID string

	// LanguageCode for Transcribe (default en-US).
	LanguageCode string

	// HTTPClient used by the thin REST helpers when SDK APIs are nil.
	HTTPClient *http.Client

	// PollyEndpoint overrides the Polly endpoint (tests).
	PollyEndpoint string

	// TranscribeEndpoint overrides a sync transcription HTTP endpoint (tests).
	TranscribeEndpoint string

	// AccessKeyID / SecretAccessKey are reserved for SDK wiring; unused by HTTP stubs.
	AccessKeyID     string
	SecretAccessKey string
}

// Client implements speech.SpeechClient.
type Client struct {
	polly      PollyAPI
	transcribe TranscribeAPI
	cfg        Config
	http       *http.Client
}

// Option configures Client.
type Option func(*Client)

// WithPolly injects a PollyAPI implementation.
func WithPolly(p PollyAPI) Option {
	return func(c *Client) { c.polly = p }
}

// WithTranscribe injects a TranscribeAPI implementation.
func WithTranscribe(t TranscribeAPI) Option {
	return func(c *Client) { c.transcribe = t }
}

// New creates an AWS speech client. Provide WithPolly / WithTranscribe for SDK
// backends, or set PollyEndpoint / TranscribeEndpoint for thin HTTP test doubles.
func New(cfg Config, opts ...Option) *Client {
	if cfg.VoiceID == "" {
		cfg.VoiceID = "Joanna"
	}
	if cfg.LanguageCode == "" {
		cfg.LanguageCode = "en-US"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	c := &Client{cfg: cfg, http: hc}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SpeechToText transcribes audio via TranscribeAPI or HTTP endpoint.
func (c *Client) SpeechToText(ctx context.Context, audio []byte) (string, error) {
	if len(audio) == 0 {
		return "", pkgerrors.InvalidArgument("audio content is required", nil)
	}
	if c.transcribe != nil {
		return c.transcribe.Transcribe(ctx, audio)
	}
	if c.cfg.TranscribeEndpoint == "" {
		return "", pkgerrors.Unimplemented("transcribe API or TranscribeEndpoint required", nil)
	}
	body, _ := json.Marshal(map[string]string{
		"audio":         base64.StdEncoding.EncodeToString(audio),
		"language_code": c.cfg.LanguageCode,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TranscribeEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return "", pkgerrors.Internal("create transcribe request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", pkgerrors.Internal("transcribe request failed", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", pkgerrors.Internal(fmt.Sprintf("transcribe HTTP %d: %s", resp.StatusCode, string(raw)), nil)
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", pkgerrors.Internal("parse transcribe response", err)
	}
	return out.Text, nil
}

// TextToSpeech synthesizes via PollyAPI or HTTP endpoint.
func (c *Client) TextToSpeech(ctx context.Context, text string, format speech.AudioFormat) ([]byte, error) {
	if text == "" {
		return nil, pkgerrors.InvalidArgument("text input is required", nil)
	}
	if format == "" {
		format = speech.FormatMP3
	}
	if c.polly != nil {
		return c.polly.SynthesizeSpeech(ctx, text, format)
	}
	if c.cfg.PollyEndpoint == "" {
		return nil, pkgerrors.Unimplemented("polly API or PollyEndpoint required", nil)
	}
	body, _ := json.Marshal(map[string]string{
		"text":          text,
		"voice_id":      c.cfg.VoiceID,
		"output_format": string(format),
		"language_code": c.cfg.LanguageCode,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.PollyEndpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, pkgerrors.Internal("create polly request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("polly request failed", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.Internal("read polly response", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, pkgerrors.Internal(fmt.Sprintf("polly HTTP %d: %s", resp.StatusCode, string(raw)), nil)
	}
	return raw, nil
}

var _ speech.SpeechClient = (*Client)(nil)
