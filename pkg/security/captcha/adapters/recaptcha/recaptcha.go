package recaptcha

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/captcha"
)

const defaultSiteVerifyURL = "https://www.google.com/recaptcha/api/siteverify"

// Config configures the reCAPTCHA siteverify client.
type Config struct {
	// SecretKey is the reCAPTCHA server secret (required).
	SecretKey string `env:"SECURITY_CAPTCHA_SECRET" validate:"required"`

	// SiteVerifyURL overrides the Google siteverify endpoint (tests/mocks).
	SiteVerifyURL string

	// MinScore is the minimum score for reCAPTCHA v3 (0–1). Zero disables the check.
	MinScore float64 `env:"SECURITY_CAPTCHA_MIN_SCORE"`

	// HTTPClient is optional; defaults to a 10s timeout client.
	HTTPClient *http.Client
}

// Verifier calls Google's reCAPTCHA siteverify API.
type Verifier struct {
	secret   string
	endpoint string
	minScore float64
	client   *http.Client
}

// Ensure Verifier implements captcha.Verifier.
var _ captcha.Verifier = (*Verifier)(nil)

type siteVerifyResponse struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score"`
	Action      string   `json:"action"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

// New creates a reCAPTCHA Verifier.
func New(cfg Config) (*Verifier, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New(captcha.CodeInvalidToken, "recaptcha secret key is required", nil)
	}
	endpoint := cfg.SiteVerifyURL
	if endpoint == "" {
		endpoint = defaultSiteVerifyURL
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Verifier{
		secret:   cfg.SecretKey,
		endpoint: endpoint,
		minScore: cfg.MinScore,
		client:   client,
	}, nil
}

// Verify posts the token to Google siteverify.
func (v *Verifier) Verify(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" {
		return captcha.ErrInvalidToken
	}

	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.New(captcha.CodeVerifyFailed, "failed to build recaptcha request", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return errors.New(captcha.CodeUnavailable, "recaptcha siteverify unreachable", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return errors.New(captcha.CodeVerifyFailed, "failed to read recaptcha response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New(captcha.CodeUnavailable, "recaptcha siteverify returned non-2xx", nil)
	}

	var result siteVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return errors.New(captcha.CodeVerifyFailed, "invalid recaptcha response", err)
	}
	if !result.Success {
		return captcha.ErrInvalidToken
	}
	if v.minScore > 0 && result.Score > 0 && result.Score < v.minScore {
		return captcha.ErrInvalidToken
	}
	return nil
}
