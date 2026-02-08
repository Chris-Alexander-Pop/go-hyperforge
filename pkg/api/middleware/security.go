package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// =========================================================================
// CSRF Protection
// =========================================================================

// CSRFConfig configures CSRF protection.
type CSRFConfig struct {
	// CookieName is the name of the CSRF cookie.
	CookieName string

	// HeaderName is the name of the header to check.
	HeaderName string

	// CookiePath is the path for the cookie.
	CookiePath string

	// CookieMaxAge is the max age of the cookie.
	CookieMaxAge int

	// Secure sets the Secure flag on the cookie.
	Secure bool

	// SameSite sets the SameSite attribute.
	SameSite http.SameSite

	// SkipPaths are paths that skip CSRF checking.
	SkipPaths []string
}

// DefaultCSRFConfig returns sensible defaults.
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		CookieName:   "csrf_token",
		HeaderName:   "X-CSRF-Token",
		CookiePath:   "/",
		CookieMaxAge: 86400, // 24 hours
		Secure:       true,
		SameSite:     http.SameSiteStrictMode,
	}
}

// CSRFProtection implements the Double Submit Cookie pattern.
// It sets a CSRF token in a cookie and expects the same token in a header.
func CSRFProtection(cfg CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip safe methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				// Set/refresh token for safe methods
				ensureCSRFToken(w, r, cfg)
				next.ServeHTTP(w, r)
				return
			}

			// Skip configured paths
			for _, path := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Validate CSRF token for unsafe methods
			cookie, err := r.Cookie(cfg.CookieName)
			if err != nil {
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			headerToken := r.Header.Get(cfg.HeaderName)
			if headerToken == "" {
				// Also check form value
				headerToken = r.FormValue(cfg.CookieName)
			}

			if headerToken == "" || headerToken != cookie.Value {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ensureCSRFToken(w http.ResponseWriter, r *http.Request, cfg CSRFConfig) {
	if _, err := r.Cookie(cfg.CookieName); err != nil {
		token := generateCSRFToken()
		http.SetCookie(w, &http.Cookie{
			Name:     cfg.CookieName,
			Value:    token,
			Path:     cfg.CookiePath,
			MaxAge:   cfg.CookieMaxAge,
			Secure:   cfg.Secure,
			HttpOnly: false, // Must be readable by JavaScript
			SameSite: cfg.SameSite,
		})
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// This should never happen in practice, but if it does, return a fallback
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.URLEncoding.EncodeToString(b)
}

// =========================================================================
// Security Headers
// =========================================================================

// SecurityHeadersConfig configures security headers.
type SecurityHeadersConfig struct {
	// HSTS
	HSTSEnabled           bool
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool

	// Content Security Policy
	CSPEnabled    bool
	CSPDirectives map[string]string

	// Other headers
	XFrameOptions       string // DENY, SAMEORIGIN
	XContentTypeOptions bool   // nosniff
	XSSProtection       bool   // Deprecated but still used
	ReferrerPolicy      string
	PermissionsPolicy   string
}

// DefaultSecurityHeadersConfig returns sensible defaults.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		HSTSEnabled:           true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false,
		CSPEnabled:            true,
		CSPDirectives: map[string]string{
			"default-src":     "'self'",
			"script-src":      "'self'",
			"style-src":       "'self' 'unsafe-inline'",
			"img-src":         "'self' data: https:",
			"font-src":        "'self'",
			"object-src":      "'none'",
			"frame-ancestors": "'none'",
			"base-uri":        "'self'",
			"form-action":     "'self'",
		},
		XFrameOptions:       "DENY",
		XContentTypeOptions: true,
		XSSProtection:       true,
		ReferrerPolicy:      "strict-origin-when-cross-origin",
	}
}

// SecurityHeaders adds security headers to responses.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// HSTS
			if cfg.HSTSEnabled {
				hsts := "max-age=" + strconv.Itoa(cfg.HSTSMaxAge)
				if cfg.HSTSIncludeSubdomains {
					hsts += "; includeSubDomains"
				}
				if cfg.HSTSPreload {
					hsts += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", hsts)
			}

			// CSP
			if cfg.CSPEnabled && len(cfg.CSPDirectives) > 0 {
				var csp strings.Builder
				for directive, value := range cfg.CSPDirectives {
					if csp.Len() > 0 {
						csp.WriteString("; ")
					}
					csp.WriteString(directive)
					csp.WriteString(" ")
					csp.WriteString(value)
				}
				w.Header().Set("Content-Security-Policy", csp.String())
			}

			// X-Frame-Options
			if cfg.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.XFrameOptions)
			}

			// X-Content-Type-Options
			if cfg.XContentTypeOptions {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// X-XSS-Protection (deprecated but still useful for older browsers)
			if cfg.XSSProtection {
				w.Header().Set("X-XSS-Protection", "1; mode=block")
			}

			// Referrer-Policy
			if cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			// Permissions-Policy
			if cfg.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// =========================================================================
// CORS
// =========================================================================

// CORSConfig configures CORS behavior.
type CORSConfig struct {
	// AllowedOrigins is the list of allowed origins.
	// Use "*" for all origins (not recommended for production).
	AllowedOrigins []string

	// AllowedMethods is the list of allowed HTTP methods.
	AllowedMethods []string

	// AllowedHeaders is the list of allowed request headers.
	AllowedHeaders []string

	// ExposedHeaders is the list of headers exposed to the client.
	ExposedHeaders []string

	// AllowCredentials indicates if credentials are allowed.
	AllowCredentials bool

	// MaxAge is the max age for preflight cache (in seconds).
	MaxAge int

	// ValidateOrigin is a custom origin validator.
	ValidateOrigin func(origin string) bool
}

// DefaultCORSConfig returns sensible defaults.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS handles Cross-Origin Resource Sharing.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowedOriginsSet := make(map[string]bool)
	for _, origin := range cfg.AllowedOrigins {
		allowedOriginsSet[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			if origin != "" {
				if len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*" {
					allowed = true
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if allowedOriginsSet[origin] {
					allowed = true
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				} else if cfg.ValidateOrigin != nil && cfg.ValidateOrigin(origin) {
					allowed = true
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				}
			}

			if !allowed && r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Handle preflight
			if r.Method == http.MethodOptions && allowed {
				// Methods
				if len(cfg.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
				}

				// Headers
				if len(cfg.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
				}

				// Credentials
				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Max-Age
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Exposed headers for actual requests
			if len(cfg.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
			}

			// Credentials for actual requests
			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			next.ServeHTTP(w, r)
		})
	}
}
