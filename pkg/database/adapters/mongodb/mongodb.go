package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// New creates a new MongoDB connection and returns the Database instance
func New(cfg database.Config) (*mongo.Database, error) {
	if cfg.Driver != database.DriverMongoDB {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for mongodb adapter", cfg.Driver), nil)
	}

	// URI construction
	uri := fmt.Sprintf("mongodb://%s:%s", cfg.Host, cfg.Port)
	if cfg.User != "" && cfg.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s", cfg.User, cfg.Password, cfg.Host, cfg.Port)
	}

	opts := options.Client().ApplyURI(uri)
	// Add timeouts
	opts.SetConnectTimeout(10 * time.Second)

	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongodb")
	}

	// Health check
	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, errors.Wrap(err, "failed to ping mongodb")
	}

	return client.Database(cfg.Name), nil
}
