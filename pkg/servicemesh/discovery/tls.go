package discovery

import (
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh"
)

// WithMTLS returns an HTTP client suitable for discovery adapters (e.g. Consul)
// using optional mutual TLS. When cfg.Enabled is false, base is returned as-is
// (or a default 15s client when base is nil).
//
// Retry/backoff for registry calls should use pkg/resilience (Retry / NewRetrier);
// this helper only configures transport security.
func WithMTLS(base *http.Client, cfg servicemesh.MTLSConfig) (*http.Client, error) {
	return cfg.HTTPClient(base)
}
