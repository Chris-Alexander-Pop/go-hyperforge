package email

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/channel"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// deliverer adapts email.Sender to channel.Deliverer.
type deliverer struct {
	sender  email.Sender
	subject string
}

func (d *deliverer) Deliver(ctx context.Context, destination, body string) error {
	return d.sender.Send(ctx, &email.Message{
		To:      []string{destination},
		Subject: d.subject,
		Body:    email.Body{PlainText: body},
		Tags:    map[string]string{"purpose": "mfa"},
	})
}

// New creates an email ChannelProvider that delivers OTPs via sender.
// Pass SendGrid/SES/SMTP email.Sender for production; memory sender for tests.
func New(sender email.Sender, cfg mfa.Config) (mfa.ChannelProvider, error) {
	if sender == nil {
		return nil, pkgerrors.InvalidArgument("email sender is required", nil)
	}
	subject := cfg.EmailSubject
	if subject == "" {
		subject = "Your verification code"
	}
	return channel.New(&deliverer{sender: sender, subject: subject}, "email", cfg)
}
