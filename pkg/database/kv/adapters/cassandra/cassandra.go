package cassandra

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/gocql/gocql"
)

const (
	defaultKeyspace = "hyperforge"
	defaultTable    = "kv"
)

// SessionAPI is the Cassandra access surface used by the adapter.
// *gocqlSession (from New) and test doubles implement this.
type SessionAPI interface {
	QueryExec(ctx context.Context, stmt string, args ...interface{}) error
	QueryScan(ctx context.Context, stmt string, args []interface{}, dest ...interface{}) error
	Close() error
}

// Config configures the Cassandra KV adapter beyond kv.Config.
type Config struct {
	kv.Config

	// Keyspace is the Cassandra keyspace (default "hyperforge").
	Keyspace string `env:"CASSANDRA_KEYSPACE" env-default:"hyperforge"`

	// Table is the KV table name (default "kv").
	Table string `env:"CASSANDRA_TABLE" env-default:"kv"`

	// Consistency is the gocql consistency level name (e.g. "QUORUM", "ONE").
	Consistency string `env:"CASSANDRA_CONSISTENCY" env-default:"LOCAL_QUORUM"`
}

// Adapter implements kv.KV for Cassandra.
type Adapter struct {
	session  SessionAPI
	keyspace string
	table    string
}

// Ensure Adapter implements kv.KV.
var _ kv.KV = (*Adapter)(nil)

// NewFromSession wraps an existing SessionAPI (production gocql wrapper or test double).
func NewFromSession(session SessionAPI, keyspace, table string) (*Adapter, error) {
	if session == nil {
		return nil, errors.InvalidArgument("cassandra session is required", nil)
	}
	if keyspace == "" {
		keyspace = defaultKeyspace
	}
	if table == "" {
		table = defaultTable
	}
	return &Adapter{session: session, keyspace: keyspace, table: table}, nil
}

// New creates a Cassandra adapter from kv/cassandra Config using gocql.
func New(cfg Config) (*Adapter, error) {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := 9042
	if cfg.Port != "" {
		p, err := strconv.Atoi(cfg.Port)
		if err != nil {
			return nil, errors.InvalidArgument("invalid cassandra port", err)
		}
		port = p
	}

	cluster := gocql.NewCluster(fmt.Sprintf("%s:%d", host, port))
	if cfg.Keyspace != "" {
		cluster.Keyspace = cfg.Keyspace
	} else {
		cluster.Keyspace = defaultKeyspace
	}
	if cfg.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: "cassandra",
			Password: cfg.Password,
		}
	}
	consistency := gocql.LocalQuorum
	if cfg.Consistency != "" {
		c, err := gocql.ParseConsistencyWrapper(cfg.Consistency)
		if err != nil {
			return nil, errors.InvalidArgument("invalid cassandra consistency", err)
		}
		consistency = c
	}
	cluster.Consistency = consistency
	cluster.Timeout = 10 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	sess, err := cluster.CreateSession()
	if err != nil {
		return nil, errors.Unavailable("failed to connect to cassandra", err)
	}

	table := cfg.Table
	if table == "" {
		table = defaultTable
	}
	return NewFromSession(&gocqlSession{sess: sess}, cluster.Keyspace, table)
}

type gocqlSession struct {
	sess *gocql.Session
}

func (s *gocqlSession) QueryExec(ctx context.Context, stmt string, args ...interface{}) error {
	return s.sess.Query(stmt, args...).WithContext(ctx).Exec()
}

func (s *gocqlSession) QueryScan(ctx context.Context, stmt string, args []interface{}, dest ...interface{}) error {
	return s.sess.Query(stmt, args...).WithContext(ctx).Scan(dest...)
}

func (s *gocqlSession) Close() error {
	s.sess.Close()
	return nil
}

func (a *Adapter) fqTable() string {
	return fmt.Sprintf("%s.%s", a.keyspace, a.table)
}

// Get retrieves a value by key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var value []byte
	stmt := fmt.Sprintf("SELECT value FROM %s WHERE key = ?", a.fqTable())
	err := a.session.QueryScan(ctx, stmt, []interface{}{key}, &value)
	if err == gocql.ErrNotFound {
		return nil, errors.NotFound("key not found", nil)
	}
	if err != nil {
		return nil, errors.Internal("cassandra get failed", err)
	}
	return value, nil
}

// Set stores a value with the given TTL (0 = no expiration).
func (a *Adapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var stmt string
	var args []interface{}
	if ttl > 0 {
		secs := int(ttl.Seconds())
		if secs < 1 {
			secs = 1
		}
		stmt = fmt.Sprintf("INSERT INTO %s (key, value) VALUES (?, ?) USING TTL ?", a.fqTable())
		args = []interface{}{key, value, secs}
	} else {
		stmt = fmt.Sprintf("INSERT INTO %s (key, value) VALUES (?, ?)", a.fqTable())
		args = []interface{}{key, value}
	}
	if err := a.session.QueryExec(ctx, stmt, args...); err != nil {
		return errors.Internal("cassandra set failed", err)
	}
	return nil
}

// Delete removes a key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stmt := fmt.Sprintf("DELETE FROM %s WHERE key = ?", a.fqTable())
	if err := a.session.QueryExec(ctx, stmt, key); err != nil {
		return errors.Internal("cassandra delete failed", err)
	}
	return nil
}

// Exists checks if a key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var found string
	stmt := fmt.Sprintf("SELECT key FROM %s WHERE key = ?", a.fqTable())
	err := a.session.QueryScan(ctx, stmt, []interface{}{key}, &found)
	if err == gocql.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, errors.Internal("cassandra exists failed", err)
	}
	return true, nil
}

// Close releases the Cassandra session.
func (a *Adapter) Close() error {
	return a.session.Close()
}
