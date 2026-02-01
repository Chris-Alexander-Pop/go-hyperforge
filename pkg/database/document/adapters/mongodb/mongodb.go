package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/document"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// New creates a new MongoDB connection and returns the Database instance
func New(cfg document.Config) (*mongo.Database, error) {
	if cfg.Driver != database.DriverMongoDB {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for mongodb adapter", cfg.Driver), nil)
	}

	// URI construction
	uri := fmt.Sprintf("mongodb://%s:%d", cfg.Host, cfg.Port)
	if cfg.User != "" && cfg.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.User, cfg.Password, cfg.Host, cfg.Port)
	}

	opts := options.Client().ApplyURI(uri)

	// Configure TLS
	if cfg.UseTLS || cfg.CAPath != "" || cfg.CertPath != "" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		}

		// Load CA Cert if provided
		if cfg.CAPath != "" {
			caCert, err := os.ReadFile(cfg.CAPath)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read CA certificate")
			}
			caCertPool := x509.NewCertPool()
			if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
				return nil, errors.New(errors.CodeInternal, "failed to append CA certificate", nil)
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Load Client Cert/Key if provided
		if cfg.CertPath != "" && cfg.KeyPath != "" {
			cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load client certificate and key")
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		opts.SetTLSConfig(tlsConfig)
	}

	// Add timeouts
	opts.SetConnectTimeout(10 * time.Second)

	// Pool Settings
	if cfg.MaxOpenConns > 0 {
		opts.SetMaxPoolSize(uint64(cfg.MaxOpenConns))
	}
	if cfg.MaxIdleConns > 0 {
		opts.SetMinPoolSize(uint64(cfg.MaxIdleConns))
	}

	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongodb")
	}

	// Health check
	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, errors.Wrap(err, "failed to ping mongodb")
	}

	return client.Database(cfg.Database), nil
}
