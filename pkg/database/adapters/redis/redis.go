package redis

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// New creates a new Redis connection
func New(cfg database.Config) (*redis.Client, error) {
	if cfg.Driver != database.DriverRedis {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for redis adapter", cfg.Driver), nil)
	}

	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password, // no password set through env implies empty usually, but we pass what we have
		DB:       0,            // default DB
	}

	// SSL Support
	if cfg.SSLMode == "require" || cfg.SSLMode == "true" {
		// Use crypto/tls for secure connection
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.SSLMode != "verify-full",
		}
	}

	client := redis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, errors.Wrap(err, "failed to ping redis")
	}

	return client, nil
}
