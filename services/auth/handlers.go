package main

import (
	"net/http"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/jwt"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/social"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/webauthn"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/labstack/echo/v4"
)

type HandlerDependencies struct {
	JWT            *jwt.Adapter
	MFA            mfa.Provider
	WebAuthn       webauthn.Service
	SocialAdapters map[string]social.Provider
	Repo           Repository
}

func BindHandlers(e *echo.Echo, deps HandlerDependencies) {
	g := e.Group("/v1/auth")

	// Base JWT Login/Verify
	g.POST("/login", handleLogin(deps))
	g.POST("/verify", handleVerify(deps))

	// MFA Operations
	g.POST("/mfa/enroll", handleMFAEnroll(deps))
	g.POST("/mfa/verify", handleMFAVerify(deps))

	// WebAuthn Operations
	g.POST("/webauthn/register/begin", handleWebAuthnRegisterBegin(deps))
	g.POST("/webauthn/register/finish", handleWebAuthnRegisterFinish(deps))
	g.POST("/webauthn/login/begin", handleWebAuthnLoginBegin(deps))
	g.POST("/webauthn/login/finish", handleWebAuthnLoginFinish(deps))

	// Social Logins
	g.GET("/social/:provider/login", handleSocialLogin(deps))
	g.GET("/social/:provider/callback", handleSocialCallback(deps))
}

/* ========================================================
	Base JWT Auth (Backed by SQLite Repository)
======================================================== */
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func handleLogin(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req LoginRequest
		if err := c.Bind(&req); err != nil {
			return errors.New(errors.CodeInvalidArgument, "invalid payload", err)
		}

		user, err := deps.Repo.GetUserByUsername(c.Request().Context(), req.Username)
		if err != nil {
			return errors.New(errors.CodeUnauthenticated, "invalid credentials", nil)
		}

		if !deps.Repo.ValidatePassword(user.PasswordHash, req.Password) {
			return errors.New(errors.CodeUnauthenticated, "invalid credentials", nil)
		}

		token, err := deps.JWT.Generate(user.ID, []string{user.Role})
		if err != nil {
			return err
		}
		
		return c.JSON(http.StatusOK, map[string]string{"token": token})
	}
}

func handleVerify(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Request().Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			return errors.New(errors.CodeUnauthenticated, "missing token", nil)
		}
		claims, err := deps.JWT.Verify(c.Request().Context(), h[7:])
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, claims)
	}
}

/* ========================================================
	Multi-Factor Authentication
======================================================== */
type MFAEnrollRequest struct {
	UserID string `json:"user_id"`
}

func handleMFAEnroll(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req MFAEnrollRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		secret, recoveryCodes, err := deps.MFA.Enroll(c.Request().Context(), req.UserID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"secret":         secret,
			"recovery_codes": recoveryCodes,
		})
	}
}

type MFAVerifyRequest struct {
	UserID string `json:"user_id"`
	Code   string `json:"code"`
}

func handleMFAVerify(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req MFAVerifyRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		
		valid, err := deps.MFA.Verify(c.Request().Context(), req.UserID, req.Code)
		if !valid || err != nil {
			valid, err = deps.MFA.Recover(c.Request().Context(), req.UserID, req.Code)
		}

		if err != nil { return err }
		if !valid { return errors.New(errors.CodeUnauthenticated, "invalid mfa code", nil) }

		return c.JSON(http.StatusOK, map[string]bool{"verified": true})
	}
}

/* ========================================================
	WebAuthn Passkeys / Biometrics
======================================================== */
type DummyUser struct {
	ID   string
	Name string
}
func (u DummyUser) WebAuthnID() []byte { return []byte(u.ID) }
func (u DummyUser) WebAuthnName() string { return u.Name }
func (u DummyUser) WebAuthnDisplayName() string { return u.Name }
func (u DummyUser) WebAuthnIcon() string { return "" }
func (u DummyUser) WebAuthnCredentials() []webauthn.Credential { return nil }

type WebAuthnReq struct {
	Username string `json:"username"`
}

func handleWebAuthnRegisterBegin(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req WebAuthnReq
		if err := c.Bind(&req); err != nil { return err }
		resp, err := deps.WebAuthn.BeginRegistration(c.Request().Context(), DummyUser{ID: req.Username, Name: req.Username})
		if err != nil { return err }
		return c.JSON(http.StatusOK, resp)
	}
}

func handleWebAuthnRegisterFinish(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req WebAuthnReq
		if err := c.Bind(&req); err != nil { return err }
		cred, err := deps.WebAuthn.FinishRegistration(c.Request().Context(), DummyUser{ID: req.Username, Name: req.Username}, nil, nil)
		if err != nil { return err }
		return c.JSON(http.StatusOK, map[string]string{"msg": "registered", "id": string(cred.ID)})
	}
}

func handleWebAuthnLoginBegin(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req WebAuthnReq
		if err := c.Bind(&req); err != nil { return err }
		resp, err := deps.WebAuthn.BeginLogin(c.Request().Context(), DummyUser{ID: req.Username, Name: req.Username})
		if err != nil { return err }
		return c.JSON(http.StatusOK, resp)
	}
}

func handleWebAuthnLoginFinish(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req WebAuthnReq
		if err := c.Bind(&req); err != nil { return err }
		cred, err := deps.WebAuthn.FinishLogin(c.Request().Context(), DummyUser{ID: req.Username, Name: req.Username}, nil, nil)
		if err != nil { return err }
		return c.JSON(http.StatusOK, map[string]string{"msg": "logged_in", "id": string(cred.ID)})
	}
}

/* ========================================================
	Social Logins
======================================================== */
func handleSocialLogin(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		provider := c.Param("provider")
		p, ok := deps.SocialAdapters[provider]
		if !ok || p == nil {
			return errors.New(errors.CodeInvalidArgument, "unsupported social provider", nil)
		}
		redirectURL := p.GetLoginURL("mock-state")
		return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

func handleSocialCallback(deps HandlerDependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		provider := c.Param("provider")
		code := c.QueryParam("code")
		p, ok := deps.SocialAdapters[provider]
		if !ok || p == nil { return errors.New(errors.CodeInvalidArgument, "unsupported provider", nil) }

		user, err := p.Exchange(c.Request().Context(), code)
		if err != nil { return errors.Wrap(err, "oauth exchange failure") }
		
		token, _ := deps.JWT.Generate(user.ID, []string{"user"})
		return c.JSON(http.StatusOK, map[string]string{"token": token, "email": user.Email})
	}
}
