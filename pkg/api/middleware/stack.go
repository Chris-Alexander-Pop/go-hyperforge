package middleware

import (
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/ratelimit"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/audit"
	auditlogger "github.com/chris-alexander-pop/go-hyperforge/pkg/audit/adapters/logger"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Config contains all security configurations.
type Config struct {
	// Security Headers
	SecurityHeaders SecurityHeadersConfig

	// CSRF
	CSRFEnabled bool
	CSRFConfig  CSRFConfig

	// CORS
	CORSEnabled bool
	CORSConfig  CORSConfig

	// Rate Limiting
	RateLimitEnabled bool
	RateLimitPerMin  int64
	RateLimitCache   cache.Cache

	// Circuit Breaker
	CircuitBreakerEnabled bool
	CircuitBreakerConfig  resilience.CircuitBreakerConfig

	// Audit Logging
	AuditConfig audit.Config
}

// DefaultConfig returns a secure default configuration.
func DefaultConfig() Config {
	return Config{
		SecurityHeaders:       DefaultSecurityHeadersConfig(),
		CSRFEnabled:           true,
		CSRFConfig:            DefaultCSRFConfig(),
		CORSEnabled:           false, // Must be explicitly enabled
		CORSConfig:            DefaultCORSConfig(),
		RateLimitEnabled:      true,
		RateLimitPerMin:       100,
		CircuitBreakerEnabled: false,
		AuditConfig: audit.Config{
			Enabled: true,
			Redact:  audit.DefaultRedactorConfig(),
		},
	}
}

// SecurityStack creates a middleware stack with all security features.
func SecurityStack(cfg Config) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		h := handler

		// Innermost: audit logging
		if cfg.AuditConfig.Enabled {
			h = AuditMiddleware(audit.New(cfg.AuditConfig, auditlogger.NewSink()))(h)
		}

		// Circuit breaker
		if cfg.CircuitBreakerEnabled {
			cb := resilience.NewCircuitBreaker(cfg.CircuitBreakerConfig)
			h = CircuitBreakerMiddleware(cb)(h)
		}

		// Rate limiting
		if cfg.RateLimitEnabled && cfg.RateLimitCache != nil {
			limiter := ratelimit.New(cfg.RateLimitCache, ratelimit.StrategySlidingWindow)
			h = RateLimitMiddleware(limiter, cfg.RateLimitPerMin, time.Minute)(h)
		}

		// CSRF protection
		if cfg.CSRFEnabled {
			h = CSRFProtection(cfg.CSRFConfig)(h)
		}

		// CORS
		if cfg.CORSEnabled {
			h = CORS(cfg.CORSConfig)(h)
		}

		// Outermost: security headers
		h = SecurityHeaders(cfg.SecurityHeaders)(h)

		return h
	}
}

// SecureDefaults applies secure defaults to an http.Server.
func SecureDefaults(srv *http.Server) {
	if srv.ReadTimeout == 0 {
		srv.ReadTimeout = 10 * time.Second
	}
	if srv.ReadHeaderTimeout == 0 {
		srv.ReadHeaderTimeout = 5 * time.Second
	}
	if srv.WriteTimeout == 0 {
		srv.WriteTimeout = 30 * time.Second
	}
	if srv.IdleTimeout == 0 {
		srv.IdleTimeout = 120 * time.Second
	}
	if srv.MaxHeaderBytes == 0 {
		srv.MaxHeaderBytes = 1 << 20 // 1 MB
	}
}
