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

// Adapter implements the document.Interface for MongoDB.
type Adapter struct {
	db     *mongo.Database
	client *mongo.Client
}

// New creates a new MongoDB connection and returns the Database instance
func New(cfg document.Config) (document.Interface, error) {
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

	return &Adapter{
		db:     client.Database(cfg.Database),
		client: client,
	}, nil
}

// Insert adds a new document to the collection.
func (a *Adapter) Insert(ctx context.Context, collection string, doc document.Document) error {
	_, err := a.db.Collection(collection).InsertOne(ctx, doc)
	if err != nil {
		return errors.Wrap(err, "failed to insert document")
	}
	return nil
}

// Find retrieves documents matching the query.
func (a *Adapter) Find(ctx context.Context, collection string, query map[string]interface{}) ([]document.Document, error) {
	cursor, err := a.db.Collection(collection).Find(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find documents")
	}
	defer cursor.Close(ctx)

	var docs []document.Document
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, errors.Wrap(err, "failed to decode documents")
	}
	return docs, nil
}

// Update modifies documents matching the filter.
func (a *Adapter) Update(ctx context.Context, collection string, filter map[string]interface{}, update map[string]interface{}) error {
	// Standardize update if not already an operator
	isOperator := false
	for k := range update {
		if len(k) > 0 && k[0] == '$' {
			isOperator = true
			break
		}
	}

	var updateDoc interface{}
	if !isOperator {
		updateDoc = map[string]interface{}{"$set": update}
	} else {
		updateDoc = update
	}

	_, err := a.db.Collection(collection).UpdateMany(ctx, filter, updateDoc)
	if err != nil {
		return errors.Wrap(err, "failed to update documents")
	}
	return nil
}

// Delete removes documents matching the filter.
func (a *Adapter) Delete(ctx context.Context, collection string, filter map[string]interface{}) error {
	_, err := a.db.Collection(collection).DeleteMany(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "failed to delete documents")
	}
	return nil
}

// Close releases resources.
func (a *Adapter) Close() error {
	if err := a.client.Disconnect(context.Background()); err != nil {
		return errors.Wrap(err, "failed to disconnect mongodb client")
	}
	return nil
}
