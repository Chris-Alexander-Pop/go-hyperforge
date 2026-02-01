package session

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

type Config struct {
	TTL time.Duration `env:"SESSION_TTL" env-default:"24h"`
}

type Adapter struct {
	store cache.Cache
	cfg   Config
}

func New(store cache.Cache, cfg Config) *Adapter {
	return &Adapter{store: store, cfg: cfg}
}

// Generate creates an opaque reference token and stores session in cache
func (a *Adapter) Generate(ctx context.Context, userID string, roles []string) (string, error) {
	token := uuid.New().String()

	// Create Claims to store
	claims := auth.Claims{
		Subject:   userID,
		Roles:     roles,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(a.cfg.TTL).Unix(),
	}

	// Store in Cache (Key: "session:{token}")
	// We rely on Cache Set to handle serialization if it was generic, but our cache expects 'db' behavior?
	// Our cache interface takes `interface{}` and handles marshalling in some adapters, but raw Set?
	// Redis adapter: Set(ctx, key, val, ttl).
	// Let's assume serialization happens in Cache adapter or here.
	// Looking at `pkg/cache/cache.go`, it takes `interface{}`, so we pass struct.

	err := a.store.Set(ctx, "session:"+token, claims, a.cfg.TTL)
	if err != nil {
		return "", errors.Wrap(err, "failed to store session")
	}

	return token, nil
}

// Verify retrieves the session from cache
func (a *Adapter) Verify(ctx context.Context, tokenString string) (*auth.Claims, error) {
	var claims auth.Claims
	err := a.store.Get(ctx, "session:"+tokenString, &claims)
	if err != nil {
		// Cache miss = Invalid/Expired
		return nil, errors.New(errors.CodeUnauthenticated, "invalid or expired session", nil)
	}

	return &claims, nil
}

// Revoke allows destroying the session immediately
func (a *Adapter) Revoke(ctx context.Context, tokenString string) error {
	return a.store.Delete(ctx, "session:"+tokenString)
}
