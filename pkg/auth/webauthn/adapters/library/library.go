package library

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/webauthn"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnService implements webauthn.Service using github.com/go-webauthn/webauthn.
type WebAuthnService struct {
	webAuthn *gowebauthn.WebAuthn
	// In a real implementation, we need a session store to hold pending ceremonies (challenges).
	// For this adapter, we will return the session data to the caller and expect them to return it to us on finish.
	// This matches the Service interface which takes 'sessionData interface{}'.
}

// New creates a new WebAuthn service.
func New(cfg webauthn.Config) (*WebAuthnService, error) {
	wconfig := &gowebauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     []string{cfg.RPOrigin},
	}

	w, err := gowebauthn.New(wconfig)
	if err != nil {
		return nil, errors.Internal("failed to initialize webauthn library", err)
	}

	return &WebAuthnService{
		webAuthn: w,
	}, nil
}

// userAdapter adapts existing webauthn.User to go-webauthn.User interface.
type userAdapter struct {
	u webauthn.User
}

func (a *userAdapter) WebAuthnID() []byte {
	return a.u.WebAuthnID()
}

func (a *userAdapter) WebAuthnName() string {
	return a.u.WebAuthnName()
}

func (a *userAdapter) WebAuthnDisplayName() string {
	return a.u.WebAuthnDisplayName()
}

func (a *userAdapter) WebAuthnIcon() string {
	return a.u.WebAuthnIcon()
}

func (a *userAdapter) WebAuthnCredentials() []gowebauthn.Credential {
	creds := a.u.WebAuthnCredentials()
	result := make([]gowebauthn.Credential, len(creds))
	for i, c := range creds {
		result[i] = gowebauthn.Credential{
			ID:              c.ID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Authenticator: gowebauthn.Authenticator{
				AAGUID:       c.Authenticator.AAGUID,
				SignCount:    c.Authenticator.SignCount,
				CloneWarning: c.Authenticator.CloneWarning,
			},
		}
	}
	return result
}

func (s *WebAuthnService) BeginRegistration(ctx context.Context, user webauthn.User) (interface{}, error) {
	// We need to convert our generic User to the library's User interface
	wUser := &userAdapter{u: user}

	// We pass empty options for now
	// The library creates the challenge
	creation, sessionData, err := s.webAuthn.BeginRegistration(wUser)
	if err != nil {
		return nil, errors.Internal("failed to begin registration", err)
	}

	// We return a map containing both the public credential creation options (to send to client)
	// and the session data (to be stored by caller/session).
	// Ideally we separate them, but the interface returns interface{}.
	// Let's assume the caller expects { "options": ..., "session": ... } or handles it.
	// But to strictly follow the interface implying it returns data for the Frontend:
	// actually `creation` is what the frontend needs. `sessionData` is what WE need later.
	// The interface signature is: BeginRegistration(...) (interface{}, error)
	// Usually this returns the JSON for the frontend.
	// BUT, we have no way to persist `sessionData` here unless we return it too.
	// Let's wrap it.

	return map[string]interface{}{
		"options": creation,
		"session": sessionData,
	}, nil
}

func (s *WebAuthnService) FinishRegistration(ctx context.Context, user webauthn.User, sessionData interface{}, responseData interface{}) (*webauthn.Credential, error) {
	wUser := &userAdapter{u: user}

	// Parse session data
	// The caller is expected to pass back the "session" part of what we returned in Begin.
	var session gowebauthn.SessionData
	if _, ok := sessionData.(map[string]interface{}); ok {
		// Try to marshall/unmarshal or manual mapping?
		// Since we returned it, let's assume the caller passes it back exactly as the library gave it if they are good citizens,
		// but across JSON boundaries it might be a map.
		// For simplicity/robustness, let's rely on JSON roundtrip if needed, or tight coupling.
		// Assuming strict Go-to-Go usage for now (e.g. HTTP handler holds it in Session store).
		// Wait, `gowebauthn.SessionData` is a struct.
		return nil, errors.InvalidArgument("session data format not supported", nil)
	} else if sd, ok := sessionData.(gowebauthn.SessionData); ok {
		session = sd
	} else {
		// Try pointer
		if sd, ok := sessionData.(*gowebauthn.SessionData); ok {
			session = *sd
		} else {
			return nil, errors.InvalidArgument("invalid session data type", nil)
		}
	}

	// Parse response data (the HTTP request)
	// The library expects an *http.Request.
	// Our interface takes `responseData interface{}`.
	// If it's an *http.Request, great.
	req, ok := responseData.(*http.Request)
	if !ok {
		return nil, errors.InvalidArgument("response data must be *http.Request", nil)
	}

	credential, err := s.webAuthn.FinishRegistration(wUser, session, req)
	if err != nil {
		return nil, errors.Conflict("registration failed", err)
	}

	return &webauthn.Credential{
		ID:              credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Authenticator: webauthn.Authenticator{
			AAGUID:       credential.Authenticator.AAGUID,
			SignCount:    credential.Authenticator.SignCount,
			CloneWarning: credential.Authenticator.CloneWarning,
		},
	}, nil
}

func (s *WebAuthnService) BeginLogin(ctx context.Context, user webauthn.User) (interface{}, error) {
	wUser := &userAdapter{u: user}

	login, sessionData, err := s.webAuthn.BeginLogin(wUser)
	if err != nil {
		return nil, errors.Internal("failed to begin login", err)
	}

	return map[string]interface{}{
		"options": login,
		"session": sessionData,
	}, nil
}

func (s *WebAuthnService) FinishLogin(ctx context.Context, user webauthn.User, sessionData interface{}, responseData interface{}) (*webauthn.Credential, error) {
	wUser := &userAdapter{u: user}

	var session gowebauthn.SessionData
	if sd, ok := sessionData.(gowebauthn.SessionData); ok {
		session = sd
	} else if sd, ok := sessionData.(*gowebauthn.SessionData); ok {
		session = *sd
	} else {
		return nil, errors.InvalidArgument("invalid session data type", nil)
	}

	req, ok := responseData.(*http.Request)
	if !ok {
		return nil, errors.InvalidArgument("response data must be *http.Request", nil)
	}

	credential, err := s.webAuthn.FinishLogin(wUser, session, req)
	if err != nil {
		return nil, errors.Unauthorized("login failed", err)
	}

	return &webauthn.Credential{
		ID:              credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Authenticator: webauthn.Authenticator{
			AAGUID:       credential.Authenticator.AAGUID,
			SignCount:    credential.Authenticator.SignCount,
			CloneWarning: credential.Authenticator.CloneWarning,
		},
	}, nil
}
