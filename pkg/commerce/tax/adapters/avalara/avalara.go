package avalara

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

const defaultBaseURL = "https://rest.avatax.com"

// Ensure compile-time interface compliance.
var _ tax.Calculator = (*Calculator)(nil)

// Config configures the Avalara AvaTax HTTP client.
type Config struct {
	// AccountID and LicenseKey authenticate via Basic auth (required).
	AccountID  string `env:"AVALARA_ACCOUNT_ID" validate:"required"`
	LicenseKey string `env:"AVALARA_LICENSE_KEY" validate:"required"`

	// CompanyCode is sent on createTransaction (default DEFAULT).
	CompanyCode string `env:"AVALARA_COMPANY_CODE" env-default:"DEFAULT"`

	// BaseURL overrides the AvaTax API root (tests via httptest).
	BaseURL string `env:"AVALARA_BASE_URL" env-default:"https://rest.avatax.com"`

	// CustomerCode is used when the caller does not supply one (orders).
	CustomerCode string `env:"AVALARA_CUSTOMER_CODE" env-default:"GUEST"`

	// RetryMaxAttempts wires pkg/resilience retries (0 disables).
	RetryMaxAttempts int           `env:"AVALARA_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"AVALARA_RETRY_BACKOFF" env-default:"100ms"`

	// HTTPClient is optional; defaults to a 15s timeout client.
	HTTPClient *http.Client
}

// Calculator implements tax.Calculator against Avalara AvaTax createTransaction.
type Calculator struct {
	accountID    string
	licenseKey   string
	companyCode  string
	customerCode string
	baseURL      string
	client       *http.Client
	retryCfg     resilience.RetryConfig
}

// New creates an Avalara calculator.
func New(cfg Config) (*Calculator, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid avalara config", err)
	}
	if strings.TrimSpace(cfg.AccountID) == "" || strings.TrimSpace(cfg.LicenseKey) == "" {
		return nil, errors.InvalidArgument("avalara account id and license key are required", nil)
	}

	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	company := cfg.CompanyCode
	if company == "" {
		company = "DEFAULT"
	}
	customer := cfg.CustomerCode
	if customer == "" {
		customer = "GUEST"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	c := &Calculator{
		accountID:    cfg.AccountID,
		licenseKey:   cfg.LicenseKey,
		companyCode:  company,
		customerCode: customer,
		baseURL:      base,
		client:       client,
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

type createTransactionRequest struct {
	Type         string                 `json:"type"`
	CompanyCode  string                 `json:"companyCode"`
	Date         string                 `json:"date"`
	CustomerCode string                 `json:"customerCode"`
	CurrencyCode string                 `json:"currencyCode"`
	Addresses    map[string]address     `json:"addresses"`
	Lines        []lineItem             `json:"lines"`
}

type address struct {
	Line1      string `json:"line1,omitempty"`
	City       string `json:"city,omitempty"`
	Region     string `json:"region,omitempty"`
	Country    string `json:"country"`
	PostalCode string `json:"postalCode,omitempty"`
}

type lineItem struct {
	Number   string  `json:"number"`
	Quantity float64 `json:"quantity"`
	Amount   float64 `json:"amount"`
	TaxCode  string  `json:"taxCode,omitempty"`
}

type createTransactionResponse struct {
	TotalTax     float64 `json:"totalTax"`
	TotalTaxable float64 `json:"totalTaxable"`
	TotalAmount  float64 `json:"totalAmount"`
	CurrencyCode string  `json:"currencyCode"`
	Summary      []struct {
		Country   string  `json:"country"`
		Region    string  `json:"region"`
		JurisType string  `json:"jurisType"`
		Rate      float64 `json:"rate"`
		Tax       float64 `json:"tax"`
	} `json:"summary"`
}

// CalculateTax calls POST /api/v2/transactions/create?$include=SummaryOnly.
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

	country := strings.ToUpper(strings.TrimSpace(loc.Country))
	region := strings.ToUpper(strings.TrimSpace(loc.State))
	reqBody := createTransactionRequest{
		Type:         "SalesOrder",
		CompanyCode:  c.companyCode,
		Date:         time.Now().UTC().Format("2006-01-02"),
		CustomerCode: c.customerCode,
		CurrencyCode: amount.Currency,
		Addresses: map[string]address{
			"singleLocation": {
				City:       strings.TrimSpace(loc.City),
				Region:     region,
				Country:    country,
				PostalCode: strings.TrimSpace(loc.PostalCode),
			},
		},
		Lines: []lineItem{{
			Number:   "1",
			Quantity: 1,
			Amount:   toMajor(amount),
			TaxCode:  "P0000000",
		}},
	}

	var out createTransactionResponse
	err := c.withRetry(ctx, func(ctx context.Context) error {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return errors.Internal("failed to marshal avalara request", err)
		}
		url := c.baseURL + "/api/v2/transactions/create?$include=SummaryOnly"
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return errors.Internal("failed to build avalara request", err)
		}
		httpReq.SetBasicAuth(c.accountID, c.licenseKey)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		resp, err := c.client.Do(httpReq)
		if err != nil {
			return errors.Unavailable("avalara unreachable", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return errors.Internal("failed to read avalara response", err)
		}
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return errors.InvalidArgument("avalara authentication failed", nil)
		}
		if resp.StatusCode == http.StatusBadRequest {
			return tax.ErrUnsupportedLocation
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.Unavailable(fmt.Sprintf("avalara returned status %d", resp.StatusCode), nil)
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return errors.Internal("invalid avalara response", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	cur := amount.Currency
	if out.CurrencyCode != "" {
		cur = strings.ToUpper(out.CurrencyCode)
	}

	breakdown := map[string]commerce.Money{}
	var combinedRate float64
	for _, s := range out.Summary {
		if s.Tax == 0 {
			continue
		}
		key := strings.ToLower(s.JurisType)
		if key == "" {
			key = "tax"
		}
		breakdown[key] = fromMajor(s.Tax, cur)
		combinedRate += s.Rate
	}

	taxable := amount
	if out.TotalTaxable != 0 {
		taxable = fromMajor(out.TotalTaxable, cur)
	}

	return &tax.TaxResult{
		TotalTax:      fromMajor(out.TotalTax, cur),
		Rate:          combinedRate,
		Breakdown:     breakdown,
		TaxableAmount: taxable,
		Jurisdiction: tax.Jurisdiction{
			Country: country,
			State:   region,
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
