// Package awsiot implements cert.CertificateProvider via an injectable AWS IoT API.
//
// Mirrors CreateKeysAndCertificate / AttachThingPrincipal / DetachThingPrincipal /
// DescribeCertificate / UpdateCertificate style operations without importing the
// AWS SDK into dependents — inject a real SDK wrapper or a test fake.
package awsiot

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/cert"
)

// Ensure Provider implements cert.CertificateProvider.
var _ cert.CertificateProvider = (*Provider)(nil)

// API is the injectable AWS IoT control-plane surface for device certificates.
type API interface {
	CreateKeysAndCertificate(ctx context.Context, setAsActive bool) (*cert.DeviceCertificate, error)
	AttachThingPrincipal(ctx context.Context, thingName, certificateARN string) error
	DetachThingPrincipal(ctx context.Context, thingName, certificateARN string) error
	DescribeCertificate(ctx context.Context, certificateID string) (*cert.DeviceCertificate, error)
	UpdateCertificate(ctx context.Context, certificateID string, status cert.CertificateStatus) error
}

// Provider implements cert.CertificateProvider over API.
type Provider struct {
	api API
}

// New creates a CertificateProvider backed by api.
func New(api API) (*Provider, error) {
	if api == nil {
		return nil, iot.ErrInvalidConfig("aws iot certificate API is required", nil)
	}
	return &Provider{api: api}, nil
}

// CreateKeysAndCertificate implements CertificateProvider.
func (p *Provider) CreateKeysAndCertificate(ctx context.Context, req cert.CreateCertificateRequest) (*cert.DeviceCertificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c, err := p.api.CreateKeysAndCertificate(ctx, req.SetAsActive)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, iot.ErrInvalidConfig("empty certificate response", nil)
	}
	if req.ThingName != "" {
		arn := c.CertificateARN
		if arn == "" {
			arn = c.CertificateID
		}
		if err := p.api.AttachThingPrincipal(ctx, req.ThingName, arn); err != nil {
			return nil, err
		}
		c.ThingName = req.ThingName
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	return c, nil
}

// AttachCertificate implements CertificateProvider.
func (p *Provider) AttachCertificate(ctx context.Context, thingName, certificateID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if thingName == "" || certificateID == "" {
		return iot.ErrInvalidConfig("thing name and certificate id are required", nil)
	}
	desc, err := p.api.DescribeCertificate(ctx, certificateID)
	if err != nil {
		return err
	}
	arn := certificateID
	if desc != nil && desc.CertificateARN != "" {
		arn = desc.CertificateARN
	}
	return p.api.AttachThingPrincipal(ctx, thingName, arn)
}

// DetachCertificate implements CertificateProvider.
func (p *Provider) DetachCertificate(ctx context.Context, thingName, certificateID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	desc, err := p.api.DescribeCertificate(ctx, certificateID)
	if err != nil {
		return err
	}
	arn := certificateID
	if desc != nil && desc.CertificateARN != "" {
		arn = desc.CertificateARN
	}
	return p.api.DetachThingPrincipal(ctx, thingName, arn)
}

// DescribeCertificate implements CertificateProvider.
func (p *Provider) DescribeCertificate(ctx context.Context, certificateID string) (*cert.DeviceCertificate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return p.api.DescribeCertificate(ctx, certificateID)
}

// UpdateCertificateStatus implements CertificateProvider.
func (p *Provider) UpdateCertificateStatus(ctx context.Context, certificateID string, status cert.CertificateStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.api.UpdateCertificate(ctx, certificateID, status)
}
