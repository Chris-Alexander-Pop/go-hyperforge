package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/otp"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// MFAProvider implements mfa.Provider using in-memory storage.
type MFAProvider struct {
	enrollments map[string]*mfa.Enrollment
	mu          *concurrency.SmartRWMutex
	totpConfig  otp.TOTPConfig
}

// New creates a new in-memory MFA provider.
func New(cfg mfa.Config) *MFAProvider {
	return &MFAProvider{
		enrollments: make(map[string]*mfa.Enrollment),
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-mfa-provider"}),
		totpConfig: otp.TOTPConfig{
			Issuer: cfg.TOTPIssuer,
			Digits: cfg.TOTPDigits,
			Period: cfg.TOTPPeriod,
		},
	}
}

func (p *MFAProvider) Enroll(ctx context.Context, userID string) (string, []string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Generate TOTP Secret
	totp := otp.NewTOTP(p.totpConfig)
	secret, err := totp.GenerateSecret()
	if err != nil {
		return "", nil, pkgerrors.Internal("failed to generate totp secret", err)
	}

	// 2. Generate Recovery Codes
	recoveryMgr := otp.NewRecoveryCodeManager(otp.DefaultRecoveryCodeConfig())
	displayCodes, hashedCodes, err := recoveryMgr.GenerateCodes()
	if err != nil {
		return "", nil, pkgerrors.Internal("failed to generate recovery codes", err)
	}

	// 3. Store Enrollment (Enabled=false until verification)
	p.enrollments[userID] = &mfa.Enrollment{
		UserID:    userID,
		Type:      "totp",
		Secret:    secret,
		Enabled:   false, // Waiting for confirmation
		Recovery:  hashedCodes,
		CreatedAt: time.Now(),
	}

	return secret, displayCodes, nil
}

func (p *MFAProvider) CompleteEnrollment(ctx context.Context, userID, code string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	enrollment, ok := p.enrollments[userID]
	if !ok {
		return pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if enrollment.Enabled {
		return pkgerrors.Conflict("mfa already enabled", nil)
	}

	totp := otp.NewTOTP(p.totpConfig)
	if !totp.Validate(enrollment.Secret, code) {
		return pkgerrors.InvalidArgument("invalid validation code", nil)
	}

	enrollment.Enabled = true
	return nil
}

func (p *MFAProvider) Verify(ctx context.Context, userID, code string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	enrollment, ok := p.enrollments[userID]
	if !ok {
		return false, pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if !enrollment.Enabled {
		return false, pkgerrors.Forbidden("mfa not enabled", nil)
	}

	totp := otp.NewTOTP(p.totpConfig)
	valid := totp.Validate(enrollment.Secret, code)

	// In a real implementation, we might want to prevent replay attacks here
	// by storing used codes or last used timestamp.
	// For memory adapter, simple validation is enough.

	return valid, nil
}

func (p *MFAProvider) Recover(ctx context.Context, userID, code string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	enrollment, ok := p.enrollments[userID]
	if !ok {
		return false, pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if !enrollment.Enabled {
		return false, pkgerrors.Forbidden("mfa not enabled", nil)
	}

	// Check recovery codes
	normalized := strings.ReplaceAll(strings.ToLower(code), "-", "")
	hash := sha256.Sum256([]byte(normalized))
	hashedCode := hex.EncodeToString(hash[:])

	for i, h := range enrollment.Recovery {
		if h == hashedCode {
			// Remove it or mark it
			enrollment.Recovery = append(enrollment.Recovery[:i], enrollment.Recovery[i+1:]...)
			return true, nil
		}
	}

	return false, nil
}

func (p *MFAProvider) Disable(ctx context.Context, userID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.enrollments[userID]; !ok {
		return pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	delete(p.enrollments, userID)
	return nil
}
