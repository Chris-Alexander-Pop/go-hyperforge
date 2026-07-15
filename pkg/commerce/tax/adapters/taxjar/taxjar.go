package taxjar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/tax"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

const defaultBaseURL = "https://api.taxjar.com"

// Ensure compile-time interface compliance.
var _ tax.Calculator = (*Calculator)(nil)

// Config configures the TaxJar HTTP client.
type Config struct {
	// APIToken is the TaxJar API token (required).
	APIToken string `env:"TAXJAR_API_TOKEN" validate:"required"`

	// BaseURL overrides the TaxJar API root (tests via httptest).
	BaseURL string `env:"TAXJAR_BASE_URL" env-default:"https://api.taxjar.com"`

	// FromCountry / FromState / FromZip identify the ship-from nexus address.
	FromCountry string `env:"TAXJAR_FROM_COUNTRY" env-default:"US"`
	FromState   string `env:"TAXJAR_FROM_STATE"`
	FromZip     string `env:"TAXJAR_FROM_ZIP"`
	FromCity    string `env:"TAXJAR_FROM_CITY"`

	// RetryMaxAttempts wires pkg/resilience retries (0 disables).
	RetryMaxAttempts int           `env:"TAXJAR_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"TAXJAR_RETRY_BACKOFF" env-default:"100ms"`

	// HTTPClient is optional; defaults to a 15s timeout client.
	HTTPClient *http.Client
}

// Calculator implements tax.Calculator against TaxJar's taxes API.
type Calculator struct {
	token      string
	baseURL    string
	from       fromAddress
	client     *http.Client
	retryCfg   resilience.RetryConfig
}

type fromAddress struct {
	country string
	state   string
	zip     string
	city    string
}

// New creates a TaxJar calculator.
func New(cfg Config) (*Calculator, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid taxjar config", err)
	}
	if strings.TrimSpace(cfg.APIToken) == "" {
		return nil, errors.InvalidArgument("taxjar api token is required", nil)
	}

	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	c := &Calculator{
		token:   cfg.APIToken,
		baseURL: base,
		from: fromAddress{
			country: strings.ToUpper(strings.TrimSpace(cfg.FromCountry)),
			state:   strings.ToUpper(strings.TrimSpace(cfg.FromState)),
			zip:     strings.TrimSpace(cfg.FromZip),
			city:    strings.TrimSpace(cfg.FromCity),
		},
		client: client,
	}
	if cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 100 * time.Millisecond
		}
		c.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf:        shouldRetry,
		}
	}
	return c, nil
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	switch err {
	case tax.ErrInvalidAmount, tax.ErrUnsupportedLocation:
		return false
	}
	return true
}

func (c *Calculator) withRetry(ctx context.Context, fn func(context.Context) error) error {
	if c.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, c.retryCfg, fn)
	}
	return fn(ctx)
}

type taxRequest struct {
	FromCountry string  `json:"from_country"`
	FromZip     string  `json:"from_zip,omitempty"`
	FromState   string  `json:"from_state,omitempty"`
	FromCity    string  `json:"from_city,omitempty"`
	ToCountry   string  `json:"to_country"`
	ToZip       string  `json:"to_zip,omitempty"`
	ToState     string  `json:"to_state,omitempty"`
	ToCity      string  `json:"to_city,omitempty"`
	Amount      float64 `json:"amount"`
	Shipping    float64 `json:"shipping"`
}

type taxResponse struct {
	Tax struct {
		OrderTotalAmount float64 `json:"order_total_amount"`
		TaxableAmount    float64 `json:"taxable_amount"`
		AmountToCollect  float64 `json:"amount_to_collect"`
		Rate             float64 `json:"rate"`
		Jurisdictions    struct {
			Country string `json:"country"`
			State   string `json:"state"`
			City    string `json:"city"`
		} `json:"jurisdictions"`
		Breakdown *struct {
			TaxCollectable      float64 `json:"tax_collectable"`
			CombinedTaxRate     float64 `json:"combined_tax_rate"`
			StateTaxCollectable float64 `json:"state_tax_collectable"`
			StateTaxRate        float64 `json:"state_tax_rate"`
			CountyTaxCollectable float64 `json:"county_tax_collectable"`
			CityTaxCollectable  float64 `json:"city_tax_collectable"`
			SpecialTaxCollectable float64 `json:"special_tax_collectable"`
			CountryTaxCollectable float64 `json:"country_tax_collectable"`
		} `json:"breakdown"`
	} `json:"tax"`
}

// CalculateTax calls POST /v2/taxes on TaxJar.
func (c *Calculator) CalculateTax(ctx context.Context, amount commerce.Money, loc tax.Location) (*tax.TaxResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := amount.Validate(); err != nil {
		return nil, tax.ErrInvalidAmount
	}
	if amount.Amount < 0 {
		return nil, tax.ErrInvalidAmount
	}
	if strings.TrimSpace(loc.Country) == "" {
		return nil, tax.ErrUnsupportedLocation
	}

	reqBody := taxRequest{
		FromCountry: c.from.country,
		FromZip:     c.from.zip,
		FromState:   c.from.state,
		FromCity:    c.from.city,
		ToCountry:   strings.ToUpper(strings.TrimSpace(loc.Country)),
		ToZip:       strings.TrimSpace(loc.PostalCode),
		ToState:     strings.ToUpper(strings.TrimSpace(loc.State)),
		ToCity:      strings.TrimSpace(loc.City),
		Amount:      toMajor(amount),
		Shipping:    0,
	}

	var out taxResponse
	err := c.withRetry(ctx, func(ctx context.Context) error {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return errors.Internal("failed to marshal taxjar request", err)
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/taxes", bytes.NewReader(payload))
		if err != nil {
			return errors.Internal("failed to build taxjar request", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		resp, err := c.client.Do(httpReq)
		if err != nil {
			return errors.Unavailable("taxjar unreachable", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return errors.Internal("failed to read taxjar response", err)
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return errors.InvalidArgument("taxjar authentication failed", nil)
		}
		if resp.StatusCode == http.StatusNotFound {
			return tax.ErrUnsupportedLocation
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.Unavailable(fmt.Sprintf("taxjar returned status %d", resp.StatusCode), nil)
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return errors.Internal("invalid taxjar response", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	cur := amount.Currency
	breakdown := map[string]commerce.Money{}
	if b := out.Tax.Breakdown; b != nil {
		if b.CountryTaxCollectable > 0 {
			breakdown["country"] = fromMajor(b.CountryTaxCollectable, cur)
		}
		if b.StateTaxCollectable > 0 {
			breakdown["state"] = fromMajor(b.StateTaxCollectable, cur)
		}
		if b.CityTaxCollectable > 0 {
			breakdown["city"] = fromMajor(b.CityTaxCollectable, cur)
		}
		if b.CountyTaxCollectable > 0 {
			breakdown["county"] = fromMajor(b.CountyTaxCollectable, cur)
		}
		if b.SpecialTaxCollectable > 0 {
			breakdown["special"] = fromMajor(b.SpecialTaxCollectable, cur)
		}
	}

	jCountry := out.Tax.Jurisdictions.Country
	if jCountry == "" {
		jCountry = reqBody.ToCountry
	}
	jState := out.Tax.Jurisdictions.State
	if jState == "" {
		jState = reqBody.ToState
	}

	taxable := amount
	if out.Tax.TaxableAmount > 0 {
		taxable = fromMajor(out.Tax.TaxableAmount, cur)
	}

	return &tax.TaxResult{
		TotalTax:      fromMajor(out.Tax.AmountToCollect, cur),
		Rate:          out.Tax.Rate,
		Breakdown:     breakdown,
		TaxableAmount: taxable,
		Jurisdiction: tax.Jurisdiction{
			Country: strings.ToUpper(jCountry),
			State:   strings.ToUpper(jState),
		},
	}, nil
}

func toMajor(m commerce.Money) float64 {
	dec := commerce.Decimals(m.Currency)
	scale := math.Pow10(dec)
	return float64(m.Amount) / scale
}

func fromMajor(amount float64, currency string) commerce.Money {
	dec := commerce.Decimals(currency)
	scale := math.Pow10(dec)
	return commerce.NewMoney(int64(math.Round(amount*scale)), currency)
}
