package awsiot_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/cert"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/cert/adapters/awsiot"
	"github.com/stretchr/testify/require"
)

type fakeAPI struct {
	mu    sync.Mutex
	certs map[string]*cert.DeviceCertificate
	seq   int
}

func newFake() *fakeAPI {
	return &fakeAPI{certs: make(map[string]*cert.DeviceCertificate)}
}

func (f *fakeAPI) CreateKeysAndCertificate(ctx context.Context, setAsActive bool) (*cert.DeviceCertificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := "cert-" + string(rune('a'+f.seq-1))
	if f.seq > 26 {
		id = "cert-n"
	}
	status := cert.StatusInactive
	if setAsActive {
		status = cert.StatusActive
	}
	c := &cert.DeviceCertificate{
		CertificateID:  id,
		CertificateARN: "arn:aws:iot:us-east-1:123:cert/" + id,
		CertificatePEM: "-----BEGIN CERTIFICATE-----\nFAKE\n-----END CERTIFICATE-----",
		PrivateKeyPEM:  "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----",
		Status:         status,
		CreatedAt:      time.Now().UTC(),
	}
	f.certs[id] = c
	cp := *c
	return &cp, nil
}

func (f *fakeAPI) AttachThingPrincipal(ctx context.Context, thingName, certificateARN string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.certs {
		if c.CertificateARN == certificateARN || c.CertificateID == certificateARN {
			c.ThingName = thingName
			return nil
		}
	}
	// Allow attach by ARN even when only ID stored in caller's copy.
	return nil
}

func (f *fakeAPI) DetachThingPrincipal(ctx context.Context, thingName, certificateARN string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.certs {
		if c.CertificateARN == certificateARN || c.CertificateID == certificateARN {
			if c.ThingName == thingName {
				c.ThingName = ""
			}
			return nil
		}
	}
	return nil
}

func (f *fakeAPI) DescribeCertificate(ctx context.Context, certificateID string) (*cert.DeviceCertificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.certs[certificateID]
	if !ok {
		return nil, errors.NotFound("certificate not found", nil)
	}
	cp := *c
	return &cp, nil
}

func (f *fakeAPI) UpdateCertificate(ctx context.Context, certificateID string, status cert.CertificateStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.certs[certificateID]
	if !ok {
		return nil
	}
	c.Status = status
	return nil
}

func TestProviderCreateAttachDescribe(t *testing.T) {
	api := newFake()
	// Fix DescribeCertificate to return proper error without missing ErrNotFound
	p, err := awsiot.New(api)
	require.NoError(t, err)

	ctx := context.Background()
	c, err := p.CreateKeysAndCertificate(ctx, cert.CreateCertificateRequest{
		ThingName:   "thing-1",
		SetAsActive: true,
	})
	require.NoError(t, err)
	require.Equal(t, cert.StatusActive, c.Status)
	require.Equal(t, "thing-1", c.ThingName)
	require.NotEmpty(t, c.CertificatePEM)

	got, err := p.DescribeCertificate(ctx, c.CertificateID)
	require.NoError(t, err)
	require.Equal(t, c.CertificateID, got.CertificateID)

	require.NoError(t, p.UpdateCertificateStatus(ctx, c.CertificateID, cert.StatusRevoked))
	got, err = p.DescribeCertificate(ctx, c.CertificateID)
	require.NoError(t, err)
	require.Equal(t, cert.StatusRevoked, got.Status)

	require.NoError(t, p.DetachCertificate(ctx, "thing-1", c.CertificateID))
	require.NoError(t, p.AttachCertificate(ctx, "thing-2", c.CertificateID))
}

func TestNewNilAPI(t *testing.T) {
	_, err := awsiot.New(nil)
	require.Error(t, err)
}
