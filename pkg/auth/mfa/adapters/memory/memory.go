package memory

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/mfa"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/mfa/otp"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
)

// MFAProvider implements mfa.Provider using in-memory storage.
type MFAProvider struct {
	enrollments map[string]*mfa.Enrollment
	mu          *concurrency.SmartRWMutex
	totpConfig  otp.TOTPConfig
	encryptor   *crypto.AESEncryptor
}

// New creates a new in-memory MFA provider.
// When cfg.EncryptionKey is set, TOTP secrets are encrypted at rest.
func New(cfg mfa.Config) (*MFAProvider, error) {
	enc, err := auth.NewAESEncryptorFromKey(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	return &MFAProvider{
		enrollments: make(map[string]*mfa.Enrollment),
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-mfa-provider"}),
		totpConfig: otp.TOTPConfig{
			Issuer: cfg.TOTPIssuer,
			Digits: cfg.TOTPDigits,
			Period: cfg.TOTPPeriod,
		},
		encryptor: enc,
	}, nil
}

func (p *MFAProvider) Enroll(ctx context.Context, userID string) (string, []string, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	totp := otp.NewTOTP(p.totpConfig)
	secret, err := totp.GenerateSecret()
	if err != nil {
		return "", nil, pkgerrors.Internal("failed to generate totp secret", err)
	}

	recoveryMgr := otp.NewRecoveryCodeManager(otp.DefaultRecoveryCodeConfig())
	displayCodes, hashedCodes, err := recoveryMgr.GenerateCodes()
	if err != nil {
		return "", nil, pkgerrors.Internal("failed to generate recovery codes", err)
	}

	storedSecret, err := p.sealSecret(secret)
	if err != nil {
		return "", nil, err
	}

	p.enrollments[userID] = &mfa.Enrollment{
		UserID:    userID,
		Type:      "totp",
		Secret:    storedSecret,
		Enabled:   false,
		Recovery:  hashedCodes,
		CreatedAt: time.Now(),
	}

	return secret, displayCodes, nil
}

func (p *MFAProvider) CompleteEnrollment(ctx context.Context, userID, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	enrollment, ok := p.enrollments[userID]
	if !ok {
		return pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if enrollment.Enabled {
		return pkgerrors.Conflict("mfa already enabled", nil)
	}

	secret, err := p.openSecret(enrollment.Secret)
	if err != nil {
		return err
	}

	totp := otp.NewTOTP(p.totpConfig)
	if !totp.Validate(secret, code) {
		return pkgerrors.InvalidArgument("invalid validation code", nil)
	}

	enrollment.Enabled = true
	return nil
}

func (p *MFAProvider) Verify(ctx context.Context, userID, code string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	enrollment, ok := p.enrollments[userID]
	if !ok {
		return false, pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if !enrollment.Enabled {
		return false, pkgerrors.Forbidden("mfa not enabled", nil)
	}

	secret, err := p.openSecret(enrollment.Secret)
	if err != nil {
		return false, err
	}

	totp := otp.NewTOTP(p.totpConfig)
	return totp.Validate(secret, code), nil
}

func (p *MFAProvider) Recover(ctx context.Context, userID, code string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	enrollment, ok := p.enrollments[userID]
	if !ok {
		return false, pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if !enrollment.Enabled {
		return false, pkgerrors.Forbidden("mfa not enabled", nil)
	}

	hashedCode := otp.HashRecoveryCode(code)
	for i, hash := range enrollment.Recovery {
		if subtle.ConstantTimeCompare([]byte(hash), []byte(hashedCode)) == 1 {
			enrollment.Recovery = append(enrollment.Recovery[:i], enrollment.Recovery[i+1:]...)
			return true, nil
		}
	}

	return false, nil
}

func (p *MFAProvider) Disable(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.enrollments[userID]; !ok {
		return pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	delete(p.enrollments, userID)
	return nil
}

func (p *MFAProvider) sealSecret(secret string) (string, error) {
	if p.encryptor == nil {
		return secret, nil
	}
	enc, err := p.encryptor.EncryptString(secret)
	if err != nil {
		return "", pkgerrors.Internal("failed to encrypt mfa secret", err)
	}
	return enc, nil
}

func (p *MFAProvider) openSecret(stored string) (string, error) {
	if p.encryptor == nil {
		return stored, nil
	}
	plain, err := p.encryptor.DecryptString(stored)
	if err != nil {
		return "", pkgerrors.Internal("failed to decrypt mfa secret", err)
	}
	return plain, nil
}
