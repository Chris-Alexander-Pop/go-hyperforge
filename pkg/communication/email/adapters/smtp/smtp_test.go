package smtp_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email/adapters/smtp"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresHost(t *testing.T) {
	_, err := smtp.New(email.Config{Driver: communication.DriverSMTP})
	require.Error(t, err)
}

func TestSendCanceledContext(t *testing.T) {
	s, err := smtp.New(email.Config{
		Driver:   communication.DriverSMTP,
		SMTPHost: "localhost",
		SMTPPort: 2525,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.Send(ctx, &email.Message{
		From:    "a@b.c",
		To:      []string{"c@d.e"},
		Subject: "x",
		Body:    email.Body{PlainText: "hi"},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSendRequiresRecipient(t *testing.T) {
	s, err := smtp.New(email.Config{
		Driver:   communication.DriverSMTP,
		SMTPHost: "localhost",
	})
	require.NoError(t, err)
	err = s.Send(context.Background(), &email.Message{From: "a@b.c", Subject: "x"})
	require.Error(t, err)
}
