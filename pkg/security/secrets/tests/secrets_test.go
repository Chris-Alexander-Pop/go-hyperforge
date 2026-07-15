package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
	secretsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type SecretsTestSuite struct {
	test.Suite
	manager secrets.SecretManager
}

func (s *SecretsTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.manager = secretsmem.New()
}

func (s *SecretsTestSuite) TestSetGet() {
	name := "db-password"
	value := "super-secret"

	err := s.manager.Set(s.Ctx, name, value)
	s.NoError(err)

	retrieved, err := s.manager.Get(s.Ctx, name)
	s.NoError(err)
	s.Equal(value, retrieved)
}

func (s *SecretsTestSuite) TestGet_NotFound() {
	_, err := s.manager.Get(s.Ctx, "unknown")
	s.Error(err)
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *SecretsTestSuite) TestRotate() {
	s.NoError(s.manager.Set(s.Ctx, "api-key", "old"))
	newVal, err := s.manager.Rotate(s.Ctx, "api-key", "new")
	s.NoError(err)
	s.Equal("new", newVal)

	got, err := s.manager.Get(s.Ctx, "api-key")
	s.NoError(err)
	s.Equal("new", got)
}

func (s *SecretsTestSuite) TestRotate_Generate() {
	s.NoError(s.manager.Set(s.Ctx, "api-key", "old"))
	newVal, err := s.manager.Rotate(s.Ctx, "api-key", "")
	s.NoError(err)
	s.NotEmpty(newVal)
	s.NotEqual("old", newVal)
}

func (s *SecretsTestSuite) TestRotate_NotFound() {
	_, err := s.manager.Rotate(s.Ctx, "missing", "x")
	s.Error(err)
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *SecretsTestSuite) TestConfigValidate() {
	cfg := secrets.DefaultConfig()
	s.Equal(security.ProviderMemory, cfg.Provider)
	s.NoError(cfg.Validate())

	bad := secrets.Config{}
	s.Error(bad.Validate())
}

func (s *SecretsTestSuite) TestEventedRotate() {
	bus := eventsmem.New(events.Config{})
	defer bus.Close()

	var gotType string
	var gotName string
	_, err := bus.Subscribe(s.Ctx, secrets.TopicSecrets, func(ctx context.Context, ev events.Event) error {
		gotType = ev.Type
		payload, ok := ev.Payload.(secrets.SecretAuditPayload)
		s.True(ok)
		gotName = payload.Name
		return nil
	})
	s.NoError(err)

	mgr := secrets.NewEventedSecretManager(secretsmem.New(), bus)
	s.NoError(mgr.Set(s.Ctx, "api-key", "v1"))
	_, err = mgr.Rotate(s.Ctx, "api-key", "v2")
	s.NoError(err)
	s.Equal(secrets.EventTypeSecretRotated, gotType)
	s.Equal("api-key", gotName)
}

func TestSecretsSuite(t *testing.T) {
	test.Run(t, new(SecretsTestSuite))
}
