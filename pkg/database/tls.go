package database

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// LoadTLSConfig creates a tls.Config based on the common DB config fields.
// It supports:
// - disable: returns nil
// - require/verify-ca/verify-full: loads CA cert, client cert/key
func LoadTLSConfig(mode, caFile, certFile, keyFile string) (*tls.Config, error) {
	if mode == "disable" || mode == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load CA if provided
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read CA cert")
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, errors.Internal("failed to append CA cert", nil)
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load Client Cert/Key if provided
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load client cert/key")
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// InsecureSkipVerify for "require" (encryption only) or testing
	// For "verify-ca" or "verify-full" we want verification.
	// Common PG modes:
	// - require: no verification, just encryption (InsecureSkipVerify = true)
	// - verify-ca: verify CA, ignore hostname
	// - verify-full: verify CA and hostname
	if mode == "require" {
		tlsConfig.InsecureSkipVerify = true
	}

	return tlsConfig, nil
}
