package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/session"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/session/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type SessionTestSuite struct {
	test.Suite
	manager session.Manager
}

func (s *SessionTestSuite) SetupTest() {
	s.Suite.SetupTest()
	mgr, err := memory.New(session.Config{TTL: time.Hour})
	s.Require().NoError(err)
	s.manager = mgr
}

func (s *SessionTestSuite) TestCreateGetDelete() {
	userID := "user-123"

	sess, err := s.manager.Create(s.Ctx, userID, nil)
	s.NoError(err)
	s.NotEmpty(sess.ID)

	got, err := s.manager.Get(s.Ctx, sess.ID)
	s.NoError(err)
	s.Equal(userID, got.UserID)

	err = s.manager.Delete(s.Ctx, sess.ID)
	s.NoError(err)

	_, err = s.manager.Get(s.Ctx, sess.ID)
	s.Error(err)
}

func (s *SessionTestSuite) TestEncryptedMetadata() {
	mgr, err := memory.New(session.Config{
		TTL:           time.Hour,
		EncryptionKey: "dev-session-encryption-passphrase",
	})
	s.Require().NoError(err)

	sess, err := mgr.Create(s.Ctx, "user-enc", map[string]interface{}{"role": "admin"})
	s.NoError(err)
	s.Equal("admin", sess.Metadata["role"])

	got, err := mgr.Get(s.Ctx, sess.ID)
	s.NoError(err)
	s.Equal("admin", got.Metadata["role"])
}

func TestSessionSuite(t *testing.T) {
	test.Run(t, new(SessionTestSuite))
}
