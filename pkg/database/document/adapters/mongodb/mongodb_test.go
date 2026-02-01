package mongodb

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/document"
	"github.com/stretchr/testify/assert"
)

func TestNew_TLSConfiguration(t *testing.T) {
	t.Run("MissingCAPath", func(t *testing.T) {
		cfg := document.Config{
			Driver:   database.DriverMongoDB,
			Host:     "localhost",
			Port:     27017,
			UseTLS:   true,
			CAPath:   "/non/existent/ca.pem",
		}
		_, err := New(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read CA certificate")
	})

	t.Run("MissingCertPath", func(t *testing.T) {
		cfg := document.Config{
			Driver:   database.DriverMongoDB,
			Host:     "localhost",
			Port:     27017,
			UseTLS:   true,
			CertPath: "/non/existent/cert.pem",
			KeyPath:  "/non/existent/key.pem",
		}
		_, err := New(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load client certificate and key")
	})
}
