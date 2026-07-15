package memory

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/webauthn"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// SessionData is returned from Begin* and must be passed back to Finish*.
type SessionData struct {
	Challenge string
	UserID    string
}

// ResponseData is the mock authenticator payload for Finish*.
// In tests, set Challenge to the SessionData.Challenge from Begin*.
type ResponseData struct {
	Challenge    string
	CredentialID []byte
	PublicKey    []byte
}

// WebAuthnService implements webauthn.Service as an in-memory test double.
// It does not perform cryptographic attestation or assertion verification.
type WebAuthnService struct {
	credentials map[string][]webauthn.Credential // userID -> credentials
	pending     map[string]SessionData           // challenge -> session
	mu          *concurrency.SmartRWMutex
	cfg         webauthn.Config
}

// New creates a new in-memory WebAuthn test service.
func New(cfg webauthn.Config) *WebAuthnService {
	return &WebAuthnService{
		credentials: make(map[string][]webauthn.Credential),
		pending:     make(map[string]SessionData),
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-webauthn"}),
		cfg:         cfg,
	}
}

func (s *WebAuthnService) BeginRegistration(ctx context.Context, user webauthn.User) (interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.InvalidArgument("user is required", nil)
	}

	challenge, err := randomChallenge()
	if err != nil {
		return nil, errors.Internal("failed to generate challenge", err)
	}

	session := SessionData{
		Challenge: challenge,
		UserID:    user.WebAuthnName(),
	}

	s.mu.Lock()
	s.pending[challenge] = session
	s.mu.Unlock()

	rpID := s.cfg.RPID
	if rpID == "" {
		rpID = "localhost"
	}

	return map[string]interface{}{
		"options": map[string]interface{}{
			"challenge": challenge,
			"rp":        map[string]string{"id": rpID, "name": s.cfg.RPDisplayName},
			"user": map[string]interface{}{
				"id":          user.WebAuthnID(),
				"name":        user.WebAuthnName(),
				"displayName": user.WebAuthnDisplayName(),
			},
		},
		"session": session,
	}, nil
}

func (s *WebAuthnService) FinishRegistration(ctx context.Context, user webauthn.User, sessionData interface{}, responseData interface{}) (*webauthn.Credential, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	session, err := parseSession(sessionData)
	if err != nil {
		return nil, err
	}
	response, err := parseResponse(responseData)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pending, ok := s.pending[session.Challenge]
	if !ok {
		return nil, errors.InvalidArgument("unknown or expired registration challenge; use adapters/library for real WebAuthn", nil)
	}
	if response.Challenge != "" && response.Challenge != pending.Challenge {
		return nil, errors.InvalidArgument("challenge mismatch; memory adapter is a test double only", nil)
	}
	delete(s.pending, session.Challenge)

	credID := response.CredentialID
	if len(credID) == 0 {
		credID = []byte("test-cred-" + session.Challenge[:8])
	}
	pubKey := response.PublicKey
	if len(pubKey) == 0 {
		pubKey = []byte("test-pubkey")
	}

	cred := webauthn.Credential{
		ID:              credID,
		PublicKey:       pubKey,
		AttestationType: "none",
	}

	userID := user.WebAuthnName()
	s.credentials[userID] = append(s.credentials[userID], cred)
	return &cred, nil
}

func (s *WebAuthnService) BeginLogin(ctx context.Context, user webauthn.User) (interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userID := user.WebAuthnName()
	creds, ok := s.credentials[userID]
	if !ok || len(creds) == 0 {
		return nil, errors.NotFound("user not found or no credentials", nil)
	}

	challenge, err := randomChallenge()
	if err != nil {
		return nil, errors.Internal("failed to generate challenge", err)
	}

	session := SessionData{
		Challenge: challenge,
		UserID:    userID,
	}
	s.pending[challenge] = session

	return map[string]interface{}{
		"options": map[string]interface{}{
			"challenge":        challenge,
			"allowCredentials": creds,
		},
		"session": session,
	}, nil
}

func (s *WebAuthnService) FinishLogin(ctx context.Context, user webauthn.User, sessionData interface{}, responseData interface{}) (*webauthn.Credential, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	session, err := parseSession(sessionData)
	if err != nil {
		return nil, err
	}
	response, err := parseResponse(responseData)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pending, ok := s.pending[session.Challenge]
	if !ok {
		return nil, errors.InvalidArgument("unknown or expired login challenge; use adapters/library for real WebAuthn", nil)
	}
	if response.Challenge != "" && response.Challenge != pending.Challenge {
		return nil, errors.InvalidArgument("challenge mismatch; memory adapter is a test double only", nil)
	}
	delete(s.pending, session.Challenge)

	userID := user.WebAuthnName()
	creds, ok := s.credentials[userID]
	if !ok || len(creds) == 0 {
		return nil, errors.NotFound("no credentials found", nil)
	}

	if len(response.CredentialID) > 0 {
		for i := range creds {
			if string(creds[i].ID) == string(response.CredentialID) {
				return &creds[i], nil
			}
		}
		return nil, errors.Unauthorized("credential not recognized", nil)
	}

	return &creds[0], nil
}

func randomChallenge() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func parseSession(sessionData interface{}) (SessionData, error) {
	switch v := sessionData.(type) {
	case SessionData:
		return v, nil
	case *SessionData:
		if v == nil {
			return SessionData{}, errors.InvalidArgument("session data is nil", nil)
		}
		return *v, nil
	case map[string]interface{}:
		if nested, ok := v["session"]; ok {
			return parseSession(nested)
		}
		challenge, _ := v["challenge"].(string)
		userID, _ := v["userID"].(string)
		if userID == "" {
			userID, _ = v["UserID"].(string)
		}
		if challenge == "" {
			return SessionData{}, errors.InvalidArgument("session data missing challenge", nil)
		}
		return SessionData{Challenge: challenge, UserID: userID}, nil
	default:
		return SessionData{}, errors.InvalidArgument(fmt.Sprintf("unsupported session data type %T; pass SessionData from Begin*", sessionData), nil)
	}
}

func parseResponse(responseData interface{}) (ResponseData, error) {
	if responseData == nil {
		// Lenient for simple tests that only check Begin*; Finish still needs a valid session.
		return ResponseData{}, nil
	}
	switch v := responseData.(type) {
	case ResponseData:
		return v, nil
	case *ResponseData:
		if v == nil {
			return ResponseData{}, nil
		}
		return *v, nil
	case map[string]interface{}:
		r := ResponseData{}
		if c, ok := v["challenge"].(string); ok {
			r.Challenge = c
		}
		if id, ok := v["credentialID"].([]byte); ok {
			r.CredentialID = id
		}
		if id, ok := v["credential_id"].(string); ok {
			r.CredentialID = []byte(id)
		}
		return r, nil
	default:
		return ResponseData{}, errors.InvalidArgument(fmt.Sprintf("unsupported response data type %T; pass ResponseData{Challenge: session.Challenge}", responseData), nil)
	}
}
