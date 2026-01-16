package cassandra

import (
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/database"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/gocql/gocql"
)

// New creates a new Cassandra session
func New(cfg database.Config) (*gocql.Session, error) {
	if cfg.Driver != database.DriverCassandra {
		return nil, errors.New(errors.CodeInvalidArgument, fmt.Sprintf("invalid driver %s for cassandra adapter", cfg.Driver), nil)
	}

	cluster := gocql.NewCluster(cfg.Host)
	cluster.Keyspace = cfg.Name // Use DB Name as Keyspace
	if cfg.User != "" && cfg.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.User,
			Password: cfg.Password,
		}
	}

	// Consistency
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to cassandra")
	}

	return session, nil
}
