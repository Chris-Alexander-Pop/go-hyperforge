package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/webauthn"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/webauthn/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type WebAuthnTestSuite struct {
	test.Suite
	service webauthn.Service
}

type mockUser struct {
	id          []byte
	name        string
	displayName string
	icon        string
	credentials []webauthn.Credential
}

func (u *mockUser) WebAuthnID() []byte                         { return u.id }
func (u *mockUser) WebAuthnName() string                       { return u.name }
func (u *mockUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *mockUser) WebAuthnIcon() string                       { return u.icon }
func (u *mockUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func (s *WebAuthnTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.service = memory.New(webauthn.Config{
		RPDisplayName: "TestApp",
		RPID:          "localhost",
		RPOrigin:      "http://localhost:8080",
	})
}

func sessionFromBegin(s *WebAuthnTestSuite, begin interface{}) memory.SessionData {
	m, ok := begin.(map[string]interface{})
	s.True(ok, "begin should return map with options+session")
	session, ok := m["session"].(memory.SessionData)
	s.True(ok, "session should be memory.SessionData")
	return session
}

func (s *WebAuthnTestSuite) TestRegistrationAndLoginFlow() {
	user := &mockUser{
		id:          []byte("user-1"),
		name:        "testuser",
		displayName: "Test User",
	}

	begin, err := s.service.BeginRegistration(s.Ctx, user)
	s.NoError(err)
	session := sessionFromBegin(s, begin)

	cred, err := s.service.FinishRegistration(s.Ctx, user, session, memory.ResponseData{
		Challenge:    session.Challenge,
		CredentialID: []byte("cred-1"),
		PublicKey:    []byte("pubkey-1"),
	})
	s.NoError(err)
	s.Equal([]byte("cred-1"), cred.ID)

	user.credentials = []webauthn.Credential{*cred}

	loginBegin, err := s.service.BeginLogin(s.Ctx, user)
	s.NoError(err)
	loginSession := sessionFromBegin(s, loginBegin)

	got, err := s.service.FinishLogin(s.Ctx, user, loginSession, memory.ResponseData{
		Challenge:    loginSession.Challenge,
		CredentialID: []byte("cred-1"),
	})
	s.NoError(err)
	s.Equal([]byte("cred-1"), got.ID)
}

func (s *WebAuthnTestSuite) TestLoginWithoutCredentials() {
	user := &mockUser{id: []byte("user-1"), name: "testuser"}
	_, err := s.service.BeginLogin(s.Ctx, user)
	s.Error(err)
}

func (s *WebAuthnTestSuite) TestChallengeMismatch() {
	user := &mockUser{id: []byte("user-1"), name: "testuser", displayName: "Test"}
	begin, err := s.service.BeginRegistration(s.Ctx, user)
	s.NoError(err)
	session := sessionFromBegin(s, begin)

	_, err = s.service.FinishRegistration(s.Ctx, user, session, memory.ResponseData{
		Challenge: "wrong-challenge",
	})
	s.Error(err)
}

func TestWebAuthnSuite(t *testing.T) {
	test.Run(t, new(WebAuthnTestSuite))
}
