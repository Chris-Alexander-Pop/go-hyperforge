// Package channel implements SMS/email MFA using a pluggable Deliverer.
//
// Prefer the typed wrappers in mfa/adapters/sms and mfa/adapters/email, which
// wire pkg/communication senders. This package is the shared in-memory store.
package channel

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/mfa"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/mfa/otp"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Deliverer sends a one-time code to a destination (phone or email).
type Deliverer interface {
	Deliver(ctx context.Context, destination, body string) error
}

type enrollment struct {
	userID      string
	channelType string
	destination string
	enabled     bool
	recovery    []string
	codeHash    string
	codeExpiry  time.Time
	createdAt   time.Time
}

// Provider is an in-memory ChannelProvider backed by a Deliverer.
type Provider struct {
	enrollments map[string]*enrollment
	mu          *concurrency.SmartRWMutex
	deliverer   Deliverer
	channelType string
	codeCfg     otp.ChannelCodeConfig
	template    string
}

// New creates a channel MFA provider.
// channelType should be "sms" or "email".
func New(deliverer Deliverer, channelType string, cfg mfa.Config) (*Provider, error) {
	if deliverer == nil {
		return nil, pkgerrors.InvalidArgument("deliverer is required", nil)
	}
	digits := cfg.CodeDigits
	if digits <= 0 {
		digits = 6
	}
	ttl := cfg.CodeTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	tmpl := cfg.MessageTemplate
	if tmpl == "" {
		tmpl = "Your verification code is %s"
	}
	return &Provider{
		enrollments: make(map[string]*enrollment),
		mu:          concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "mfa-channel-" + channelType}),
		deliverer:   deliverer,
		channelType: channelType,
		codeCfg: otp.ChannelCodeConfig{
			Digits: digits,
			TTL:    ttl,
		},
		template: tmpl,
	}, nil
}

func (p *Provider) Enroll(ctx context.Context, userID, destination string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if userID == "" || destination == "" {
		return nil, pkgerrors.InvalidArgument("userID and destination are required", nil)
	}

	recoveryMgr := otp.NewRecoveryCodeManager(otp.DefaultRecoveryCodeConfig())
	displayCodes, hashedCodes, err := recoveryMgr.GenerateCodes()
	if err != nil {
		return nil, pkgerrors.Internal("failed to generate recovery codes", err)
	}

	code, err := otp.GenerateChannelCode(p.codeCfg)
	if err != nil {
		return nil, pkgerrors.Internal("failed to generate challenge code", err)
	}

	p.mu.Lock()
	p.enrollments[userID] = &enrollment{
		userID:      userID,
		channelType: p.channelType,
		destination: destination,
		enabled:     false,
		recovery:    hashedCodes,
		codeHash:    otp.HashChannelCode(code),
		codeExpiry:  time.Now().Add(p.codeCfg.TTL),
		createdAt:   time.Now(),
	}
	p.mu.Unlock()

	body := fmt.Sprintf(p.template, code)
	if err := p.deliverer.Deliver(ctx, destination, body); err != nil {
		p.mu.Lock()
		delete(p.enrollments, userID)
		p.mu.Unlock()
		return nil, pkgerrors.Wrap(err, "failed to deliver enrollment challenge")
	}

	return displayCodes, nil
}

func (p *Provider) CompleteEnrollment(ctx context.Context, userID, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.enrollments[userID]
	if !ok {
		return pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if e.enabled {
		return pkgerrors.Conflict("mfa already enabled", nil)
	}
	if !p.matchCode(e, code) {
		return pkgerrors.InvalidArgument("invalid validation code", nil)
	}
	e.enabled = true
	e.codeHash = ""
	e.codeExpiry = time.Time{}
	return nil
}

func (p *Provider) SendChallenge(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	code, err := otp.GenerateChannelCode(p.codeCfg)
	if err != nil {
		return pkgerrors.Internal("failed to generate challenge code", err)
	}

	p.mu.Lock()
	e, ok := p.enrollments[userID]
	if !ok {
		p.mu.Unlock()
		return pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if !e.enabled {
		p.mu.Unlock()
		return pkgerrors.Forbidden("mfa not enabled", nil)
	}
	destination := e.destination
	e.codeHash = otp.HashChannelCode(code)
	e.codeExpiry = time.Now().Add(p.codeCfg.TTL)
	p.mu.Unlock()

	body := fmt.Sprintf(p.template, code)
	if err := p.deliverer.Deliver(ctx, destination, body); err != nil {
		return pkgerrors.Wrap(err, "failed to deliver challenge")
	}
	return nil
}

func (p *Provider) Verify(ctx context.Context, userID, code string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.enrollments[userID]
	if !ok {
		return false, pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if !e.enabled {
		return false, pkgerrors.Forbidden("mfa not enabled", nil)
	}
	if !p.matchCode(e, code) {
		return false, nil
	}
	e.codeHash = ""
	e.codeExpiry = time.Time{}
	return true, nil
}

func (p *Provider) Recover(ctx context.Context, userID, code string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	e, ok := p.enrollments[userID]
	if !ok {
		return false, pkgerrors.NotFound("mfa enrollment not found", nil)
	}
	if !e.enabled {
		return false, pkgerrors.Forbidden("mfa not enabled", nil)
	}

	hashedCode := otp.HashRecoveryCode(code)
	for i, hash := range e.recovery {
		if subtle.ConstantTimeCompare([]byte(hash), []byte(hashedCode)) == 1 {
			e.recovery = append(e.recovery[:i], e.recovery[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (p *Provider) Disable(ctx context.Context, userID string) error {
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

func (p *Provider) matchCode(e *enrollment, code string) bool {
	if e.codeHash == "" || time.Now().After(e.codeExpiry) {
		return false
	}
	got := otp.HashChannelCode(code)
	return subtle.ConstantTimeCompare([]byte(e.codeHash), []byte(got)) == 1
}

// Destination returns the enrolled destination for tests/introspection.
func (p *Provider) Destination(userID string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.enrollments[userID]
	if !ok {
		return "", false
	}
	return e.destination, true
}
