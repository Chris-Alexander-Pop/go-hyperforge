package sms

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/channel"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// deliverer adapts sms.Sender to channel.Deliverer.
type deliverer struct {
	sender sms.Sender
}

func (d *deliverer) Deliver(ctx context.Context, destination, body string) error {
	return d.sender.Send(ctx, &sms.Message{
		To:   destination,
		Body: body,
		Tags: map[string]string{"purpose": "mfa"},
	})
}

// New creates an SMS ChannelProvider that delivers OTPs via sender.
// Pass a Twilio (or SNS) sms.Sender for production; memory sender for tests.
func New(sender sms.Sender, cfg mfa.Config) (mfa.ChannelProvider, error) {
	if sender == nil {
		return nil, pkgerrors.InvalidArgument("sms sender is required", nil)
	}
	return channel.New(&deliverer{sender: sender}, "sms", cfg)
}
