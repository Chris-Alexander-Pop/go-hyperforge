package openexchangerates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/currency"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

const (
	defaultBaseURL = "https://openexchangerates.org/api"
	// frankfurterBaseURL is a free, keyless alternative with a compatible rates map.
	// Set BaseURL to this (or httptest) when AppID is empty for local/dev use.
	frankfurterBaseURL = "https://api.frankfurter.app"
)

// Ensure compile-time interface compliance.
var (
	_ currency.Converter        = (*Provider)(nil)
	_ currency.LiveRateProvider = (*Provider)(nil)
)

// Config configures the Open Exchange Rates (or Frankfurter-compatible) client.
type Config struct {
	// AppID is the Open Exchange Rates app_id. Optional when using Frankfurter
	// (BaseURL pointing at api.frankfurter.app) or an httptest stub.
	AppID string `env:"OPENEXCHANGERATES_APP_ID"`

	// BaseURL overrides the API root. Defaults to Open Exchange Rates.
	// For a free feed without AppID, set to https://api.frankfurter.app.
	BaseURL string `env:"OPENEXCHANGERATES_BASE_URL" env-default:"https://openexchangerates.org/api"`

	// DefaultBase is the FX base currency when FetchRates base is empty.
	DefaultBase string `env:"OPENEXCHANGERATES_BASE" env-default:"USD"`

	// CacheTTL controls optional rate caching via Cache (0 disables caching).
	CacheTTL time.Duration `env:"OPENEXCHANGERATES_CACHE_TTL" env-default:"1h"`

	// Cache is optional; when set with CacheTTL > 0, FetchRates results are cached.
	Cache cache.Cache

	// RetryMaxAttempts wires pkg/resilience retries (0 disables).
	RetryMaxAttempts int           `env:"OPENEXCHANGERATES_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"OPENEXCHANGERATES_RETRY_BACKOFF" env-default:"100ms"`

	// HTTPClient is optional; defaults to a 15s timeout client.
	HTTPClient *http.Client
}

// Provider implements currency.Converter and currency.LiveRateProvider.
type Provider struct {
	appID       string
	baseURL     string
	defaultBase string
	cache       cache.Cache
	cacheTTL    time.Duration
	client      *http.Client
	retryCfg    resilience.RetryConfig
	rates       map[string]float64 // last successful fetch, keyed relative to lastBase
	lastBase    string
	mu          *concurrency.SmartRWMutex
}

// New creates a live FX provider.
func New(cfg Config) (*Provider, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid openexchangerates config", err)
	}

	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	// Open Exchange Rates requires an app id; Frankfurter (and httptest) do not.
	needsAppID := strings.Contains(base, "openexchangerates.org")
	if needsAppID && strings.TrimSpace(cfg.AppID) == "" {
		return nil, errors.InvalidArgument("openexchangerates app id is required", nil)
	}

	defaultBase := strings.ToUpper(strings.TrimSpace(cfg.DefaultBase))
	if defaultBase == "" {
		defaultBase = "USD"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	p := &Provider{
		appID:       cfg.AppID,
		baseURL:     base,
		defaultBase: defaultBase,
		cache:       cfg.Cache,
		cacheTTL:    cfg.CacheTTL,
		client:      client,
		rates:       map[string]float64{defaultBase: 1.0},
		lastBase:    defaultBase,
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "commerce-fx-openexchangerates"}),
	}
	if cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 100 * time.Millisecond
		}
		p.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				switch err {
				case currency.ErrUnsupportedCurrency, currency.ErrInvalidAmount, currency.ErrSameCurrency:
					return false
				}
				return true
			},
		}
	}
	return p, nil
}

// NewFrankfurter creates a provider pointed at the free Frankfurter API (no AppID).
func NewFrankfurter(opts ...func(*Config)) (*Provider, error) {
	cfg := Config{
		BaseURL:          frankfurterBaseURL,
		DefaultBase:      "USD",
		CacheTTL:         time.Hour,
		RetryMaxAttempts: 3,
		RetryBackoff:     100 * time.Millisecond,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return New(cfg)
}

func (p *Provider) withRetry(ctx context.Context, fn func(context.Context) error) error {
	if p.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, p.retryCfg, fn)
	}
	return fn(ctx)
}

type oerLatestResponse struct {
	Base      string             `json:"base"`
	Timestamp int64              `json:"timestamp"`
	Rates     map[string]float64 `json:"rates"`
}

type frankfurterResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// FetchRates returns rates keyed by currency code relative to base.
func (p *Provider) FetchRates(ctx context.Context, base string) (map[string]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		base = p.defaultBase
	}

	cacheKey := "fx:rates:" + base
	if p.cache != nil && p.cacheTTL > 0 {
		var cached map[string]float64
		if err := p.cache.Get(ctx, cacheKey, &cached); err == nil && len(cached) > 0 {
			p.storeRates(base, cached)
			return copyRates(cached), nil
		}
	}

	var rates map[string]float64
	err := p.withRetry(ctx, func(ctx context.Context) error {
		var e error
		rates, e = p.fetchHTTP(ctx, base)
		return e
	})
	if err != nil {
		return nil, err
	}

	p.storeRates(base, rates)
	if p.cache != nil && p.cacheTTL > 0 {
		_ = p.cache.Set(ctx, cacheKey, rates, p.cacheTTL)
	}
	return copyRates(rates), nil
}

func (p *Provider) fetchHTTP(ctx context.Context, base string) (map[string]float64, error) {
	reqURL, err := p.buildLatestURL(base)
	if err != nil {
		return nil, errors.Internal("failed to build fx url", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Internal("failed to build fx request", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, errors.Unavailable("fx provider unreachable", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, errors.Internal("failed to read fx response", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.Unavailable(fmt.Sprintf("fx provider returned status %d", resp.StatusCode), nil)
	}

	rates, err := parseRatesResponse(body, base)
	if err != nil {
		return nil, err
	}
	return rates, nil
}

func (p *Provider) buildLatestURL(base string) (string, error) {
	// Open Exchange Rates when AppID is set or the host is OER.
	useOER := p.appID != "" || strings.Contains(p.baseURL, "openexchangerates")
	if useOER {
		u, err := url.Parse(p.baseURL + "/latest.json")
		if err != nil {
			return "", err
		}
		q := u.Query()
		q.Set("app_id", p.appID)
		q.Set("base", base)
		u.RawQuery = q.Encode()
		return u.String(), nil
	}
	// Frankfurter-style (free, keyless): GET /latest?from=USD
	u, err := url.Parse(p.baseURL + "/latest")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("from", base)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseRatesResponse(body []byte, fallbackBase string) (map[string]float64, error) {
	var oer oerLatestResponse
	if err := json.Unmarshal(body, &oer); err == nil && len(oer.Rates) > 0 {
		base := strings.ToUpper(oer.Base)
		if base == "" {
			base = fallbackBase
		}
		rates := make(map[string]float64, len(oer.Rates)+1)
		for k, v := range oer.Rates {
			rates[strings.ToUpper(k)] = v
		}
		rates[base] = 1.0
		return rates, nil
	}
	var fr frankfurterResponse
	if err := json.Unmarshal(body, &fr); err != nil {
		return nil, errors.Internal("invalid fx response", err)
	}
	if len(fr.Rates) == 0 {
		return nil, currency.ErrLiveRatesUnavailable
	}
	base := strings.ToUpper(fr.Base)
	if base == "" {
		base = fallbackBase
	}
	rates := make(map[string]float64, len(fr.Rates)+1)
	for k, v := range fr.Rates {
		rates[strings.ToUpper(k)] = v
	}
	rates[base] = 1.0
	return rates, nil
}

func (p *Provider) storeRates(base string, rates map[string]float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastBase = base
	p.rates = copyRates(rates)
}

func copyRates(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// GetRate returns the exchange rate between two currencies, refreshing if needed.
func (p *Provider) GetRate(ctx context.Context, from string, to string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == "" || to == "" {
		return 0, currency.ErrUnsupportedCurrency
	}
	if from == to {
		return 1.0, nil
	}

	p.mu.RLock()
	r1, ok1 := p.rates[from]
	r2, ok2 := p.rates[to]
	base := p.lastBase
	p.mu.RUnlock()

	if !ok1 || !ok2 {
		// Refresh relative to from (or default) so both codes appear.
		refreshBase := from
		if base != "" {
			refreshBase = base
		}
		if _, err := p.FetchRates(ctx, refreshBase); err != nil {
			return 0, err
		}
		p.mu.RLock()
		r1, ok1 = p.rates[from]
		r2, ok2 = p.rates[to]
		p.mu.RUnlock()
		if !ok1 || !ok2 {
			// Try fetching with from as base directly.
			if _, err := p.FetchRates(ctx, from); err != nil {
				return 0, err
			}
			p.mu.RLock()
			r1, ok1 = p.rates[from]
			r2, ok2 = p.rates[to]
			p.mu.RUnlock()
		}
	}
	if !ok1 || !ok2 {
		return 0, currency.ErrUnsupportedCurrency
	}
	if r1 == 0 {
		return 0, currency.ErrLiveRatesUnavailable
	}
	return r2 / r1, nil
}

// Convert converts an amount using live (or cached) rates.
func (p *Provider) Convert(ctx context.Context, amount float64, from string, to string) (*currency.ConversionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, currency.ErrInvalidAmount
	}
	rate, err := p.GetRate(ctx, from, to)
	if err != nil {
		return nil, err
	}
	return &currency.ConversionResult{
		FromAmount: amount,
		From:       strings.ToUpper(from),
		ToAmount:   amount * rate,
		To:         strings.ToUpper(to),
		Rate:       rate,
		Timestamp:  time.Now().UTC(),
	}, nil
}
