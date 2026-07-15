// Package cert provides device certificate helper types for IoT provisioning.
//
// These types model X.509 device identity material (PEM + metadata) and a
// CertificateProvider interface for create/attach/detach flows. Cloud SDK
// wiring (AWS IoT CreateKeysAndCertificate, etc.) is left to adapters; the
// memory provider supports local tests and scaffolding.
package cert

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
)

// DeviceCertificate holds PEM-encoded device identity material.
type DeviceCertificate struct {
	// CertificateID is a provider-assigned identifier.
	CertificateID string

	// CertificateARN is an optional cloud ARN / resource name.
	CertificateARN string

	// CertificatePEM is the X.509 certificate in PEM form.
	CertificatePEM string

	// PrivateKeyPEM is the private key in PEM form (may be empty when CSR-based).
	PrivateKeyPEM string

	// PublicKeyPEM is optional public key material.
	PublicKeyPEM string

	// Status is Active, Inactive, Revoked, or PendingTransfer.
	Status CertificateStatus

	// CreatedAt is when the certificate was issued.
	CreatedAt time.Time

	// ExpiresAt is optional expiry.
	ExpiresAt *time.Time

	// ThingName is the device/thing this cert is attached to (if any).
	ThingName string
}

// CertificateStatus is the lifecycle state of a device certificate.
type CertificateStatus string

const (
	StatusActive          CertificateStatus = "ACTIVE"
	StatusInactive        CertificateStatus = "INACTIVE"
	StatusRevoked         CertificateStatus = "REVOKED"
	StatusPendingTransfer CertificateStatus = "PENDING_TRANSFER"
)

// CreateCertificateRequest configures certificate creation.
type CreateCertificateRequest struct {
	// ThingName optionally attaches the cert to a thing after creation.
	ThingName string

	// SetAsActive marks the certificate active immediately.
	SetAsActive bool
}

// CertificateProvider provisions and manages device certificates.
type CertificateProvider interface {
	// CreateKeysAndCertificate creates a new key pair + certificate.
	CreateKeysAndCertificate(ctx context.Context, req CreateCertificateRequest) (*DeviceCertificate, error)

	// AttachCertificate associates a certificate with a thing/device.
	AttachCertificate(ctx context.Context, thingName, certificateID string) error

	// DetachCertificate removes a certificate association.
	DetachCertificate(ctx context.Context, thingName, certificateID string) error

	// DescribeCertificate returns certificate metadata.
	DescribeCertificate(ctx context.Context, certificateID string) (*DeviceCertificate, error)

	// UpdateCertificateStatus activates, deactivates, or revokes a certificate.
	UpdateCertificateStatus(ctx context.Context, certificateID string, status CertificateStatus) error
}

// MemoryProvider is an in-memory CertificateProvider for tests.
type MemoryProvider struct {
	mu    *concurrency.SmartRWMutex
	certs map[string]*DeviceCertificate
}

// NewMemoryProvider creates an empty in-memory certificate store.
func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "iot-device-cert"}),
		certs: make(map[string]*DeviceCertificate),
	}
}

// CreateKeysAndCertificate implements CertificateProvider.
func (p *MemoryProvider) CreateKeysAndCertificate(ctx context.Context, req CreateCertificateRequest) (*DeviceCertificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, err := randomID()
	if err != nil {
		return nil, iot.ErrInvalidConfig("failed to generate certificate id", err)
	}
	status := StatusInactive
	if req.SetAsActive {
		status = StatusActive
	}
	cert := &DeviceCertificate{
		CertificateID:  id,
		CertificateARN: "arn:memory:iot:cert/" + id,
		CertificatePEM: "-----BEGIN CERTIFICATE-----\nMEMORY\n-----END CERTIFICATE-----",
		PrivateKeyPEM:  "-----BEGIN PRIVATE KEY-----\nMEMORY\n-----END PRIVATE KEY-----",
		PublicKeyPEM:   "-----BEGIN PUBLIC KEY-----\nMEMORY\n-----END PUBLIC KEY-----",
		Status:         status,
		CreatedAt:      time.Now().UTC(),
		ThingName:      req.ThingName,
	}
	p.mu.Lock()
	p.certs[id] = cert
	p.mu.Unlock()
	cp := *cert
	return &cp, nil
}

// AttachCertificate implements CertificateProvider.
func (p *MemoryProvider) AttachCertificate(ctx context.Context, thingName, certificateID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if thingName == "" || certificateID == "" {
		return iot.ErrInvalidConfig("thing name and certificate id are required", nil)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := p.certs[certificateID]
	if !ok {
		return iot.ErrInvalidConfig("certificate not found", nil)
	}
	c.ThingName = thingName
	return nil
}

// DetachCertificate implements CertificateProvider.
func (p *MemoryProvider) DetachCertificate(ctx context.Context, thingName, certificateID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := p.certs[certificateID]
	if !ok {
		return iot.ErrInvalidConfig("certificate not found", nil)
	}
	if c.ThingName == thingName {
		c.ThingName = ""
	}
	return nil
}

// DescribeCertificate implements CertificateProvider.
func (p *MemoryProvider) DescribeCertificate(ctx context.Context, certificateID string) (*DeviceCertificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	c, ok := p.certs[certificateID]
	if !ok {
		return nil, iot.ErrInvalidConfig("certificate not found", nil)
	}
	cp := *c
	return &cp, nil
}

// UpdateCertificateStatus implements CertificateProvider.
func (p *MemoryProvider) UpdateCertificateStatus(ctx context.Context, certificateID string, status CertificateStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := p.certs[certificateID]
	if !ok {
		return iot.ErrInvalidConfig("certificate not found", nil)
	}
	c.Status = status
	return nil
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

var _ CertificateProvider = (*MemoryProvider)(nil)
