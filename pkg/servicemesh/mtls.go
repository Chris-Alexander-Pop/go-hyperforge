package servicemesh

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// MTLSConfig holds mutual TLS settings for mesh client/server connections.
//
// This package provides config types and helpers only — it is not a full service
// mesh data plane (no Envoy/Linkerd sidecar, no automatic traffic policy).
type MTLSConfig struct {
	// Enabled turns on TLS for outbound discovery/HTTP clients.
	Enabled bool `env:"MESH_MTLS_ENABLED" env-default:"false"`

	// CertFile is the client (or server) certificate PEM path.
	CertFile string `env:"MESH_MTLS_CERT_FILE"`

	// KeyFile is the private key PEM path matching CertFile.
	KeyFile string `env:"MESH_MTLS_KEY_FILE"`

	// CAFile is the PEM CA bundle used to verify peer certificates.
	CAFile string `env:"MESH_MTLS_CA_FILE"`

	// ServerName overrides the TLS server name for verification (SNI).
	ServerName string `env:"MESH_MTLS_SERVER_NAME"`

	// InsecureSkipVerify disables peer verification (dev only).
	InsecureSkipVerify bool `env:"MESH_MTLS_INSECURE" env-default:"false"`

	// MinVersion is the minimum TLS version (default TLS 1.2).
	MinVersion uint16
}

// TLSConfig builds a *tls.Config from MTLSConfig. Returns nil when Enabled is false.
func (c MTLSConfig) TLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.InsecureSkipVerify, //nolint:gosec // explicit opt-in for tests/dev
		ServerName:         c.ServerName,
	}
	if c.MinVersion != 0 {
		cfg.MinVersion = c.MinVersion
	}
	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("servicemesh: load key pair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	if c.CAFile != "" {
		pem, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("servicemesh: read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("servicemesh: no certificates found in CA file")
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}

// HTTPClient returns an *http.Client whose transport uses c when Enabled.
// When TLS is disabled, base is returned unchanged (or a default client).
func (c MTLSConfig) HTTPClient(base *http.Client) (*http.Client, error) {
	tlsCfg, err := c.TLSConfig()
	if err != nil {
		return nil, err
	}
	if base == nil {
		base = &http.Client{Timeout: 15 * time.Second}
	}
	if tlsCfg == nil {
		return base, nil
	}

	client := *base
	var transport *http.Transport
	if base.Transport != nil {
		if t, ok := base.Transport.(*http.Transport); ok {
			transport = t.Clone()
		}
	}
	if transport == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	transport.TLSClientConfig = tlsCfg
	client.Transport = transport
	return &client, nil
}

// DialTLS dials network/address with optional mTLS. When cfg.Enabled is false,
// it uses a plain net.Dialer.
func DialTLS(network, address string, cfg MTLSConfig, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	d := net.Dialer{Timeout: timeout}
	tlsCfg, err := cfg.TLSConfig()
	if err != nil {
		return nil, err
	}
	if tlsCfg == nil {
		return d.Dial(network, address)
	}
	return tls.DialWithDialer(&d, network, address, tlsCfg)
}
