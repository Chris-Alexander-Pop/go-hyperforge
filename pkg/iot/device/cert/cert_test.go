package cert_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/cert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryProvider_Lifecycle(t *testing.T) {
	p := cert.NewMemoryProvider()
	ctx := context.Background()

	c, err := p.CreateKeysAndCertificate(ctx, cert.CreateCertificateRequest{
		ThingName:   "device-1",
		SetAsActive: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, c.CertificateID)
	assert.Equal(t, cert.StatusActive, c.Status)
	assert.Contains(t, c.CertificatePEM, "BEGIN CERTIFICATE")

	got, err := p.DescribeCertificate(ctx, c.CertificateID)
	require.NoError(t, err)
	assert.Equal(t, "device-1", got.ThingName)

	require.NoError(t, p.DetachCertificate(ctx, "device-1", c.CertificateID))
	require.NoError(t, p.AttachCertificate(ctx, "device-2", c.CertificateID))
	got, err = p.DescribeCertificate(ctx, c.CertificateID)
	require.NoError(t, err)
	assert.Equal(t, "device-2", got.ThingName)

	require.NoError(t, p.UpdateCertificateStatus(ctx, c.CertificateID, cert.StatusRevoked))
	got, err = p.DescribeCertificate(ctx, c.CertificateID)
	require.NoError(t, err)
	assert.Equal(t, cert.StatusRevoked, got.Status)
}
