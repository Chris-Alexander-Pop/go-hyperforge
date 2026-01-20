package sendgrid

import (
	"context"
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Sender implements email.Sender for SendGrid.
type Sender struct {
	apiKey string
}

// New creates a new SendGrid sender.
func New(cfg email.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.SendGridAPIKey == "" {
		return nil, errors.InvalidArgument("SendGrid API key is required", nil)
	}

	return &Sender{
		apiKey: cfg.SendGridAPIKey,
	}, nil
}

// Send implements email.Sender.
func (s *Sender) Send(ctx context.Context, msg *email.Message) error {
	m := mail.NewV3Mail()

	fromEmail := msg.From
	if fromEmail == "" {
		// In a real scenario, we might fallback to a default from config
		// For now, allow SendGrid to validation error if empty
	}
	m.SetFrom(mail.NewEmail("", fromEmail))

	p := mail.NewPersonalization()
	for _, to := range msg.To {
		p.AddTos(mail.NewEmail("", to))
	}
	for _, cc := range msg.CC {
		p.AddCCs(mail.NewEmail("", cc))
	}
	for _, bcc := range msg.BCC {
		p.AddBCCs(mail.NewEmail("", bcc))
	}
	m.AddPersonalizations(p)

	m.Subject = msg.Subject

	if msg.Body.PlainText != "" {
		m.AddContent(mail.NewContent("text/plain", msg.Body.PlainText))
	}
	if msg.Body.HTML != "" {
		m.AddContent(mail.NewContent("text/html", msg.Body.HTML))
	}

	client := sendgrid.NewSendClient(s.apiKey)
	resp, err := client.Send(m)
	if err != nil {
		return errors.Internal("failed to send email via sendgrid", err)
	}

	if resp.StatusCode >= 400 {
		return errors.Internal("sendgrid api error", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, resp.Body))
	}

	return nil
}

// SendBatch implements email.Sender.
func (s *Sender) SendBatch(ctx context.Context, msgs []*email.Message) error {
	for _, msg := range msgs {
		if err := s.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

// Close implements email.Sender.
func (s *Sender) Close() error {
	return nil
}
