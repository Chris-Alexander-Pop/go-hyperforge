package sendgrid_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email/adapters/sendgrid"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresAPIKey(t *testing.T) {
	_, err := sendgrid.New(email.Config{Driver: communication.DriverSendGrid})
	require.Error(t, err)
}

func TestSendCanceledContext(t *testing.T) {
	s, err := sendgrid.New(email.Config{
		Driver:         communication.DriverSendGrid,
		SendGridAPIKey: "SG.test",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.Send(ctx, &email.Message{To: []string{"a@b.c"}, Subject: "x", Body: email.Body{PlainText: "hi"}})
	require.ErrorIs(t, err, context.Canceled)
}
