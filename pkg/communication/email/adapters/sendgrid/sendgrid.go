package sendgrid

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Sender implements email.Sender for SendGrid.
type Sender struct {
	client      *sendgrid.Client
	defaultFrom string
}

// New creates a new SendGrid sender.
func New(cfg email.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.SendGridAPIKey == "" {
		return nil, errors.InvalidArgument("SendGrid API key is required", nil)
	}

	return &Sender{
		client:      sendgrid.NewSendClient(cfg.SendGridAPIKey),
		defaultFrom: cfg.DefaultFrom,
	}, nil
}

// Send implements email.Sender.
func (s *Sender) Send(ctx context.Context, msg *email.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if msg == nil {
		return errors.InvalidArgument("message is required", nil)
	}
	if len(msg.To) == 0 {
		return errors.InvalidArgument("at least one recipient is required", nil)
	}

	m := mail.NewV3Mail()

	fromEmail := msg.From
	if fromEmail == "" {
		fromEmail = s.defaultFrom
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

	if msg.ReplyTo != "" {
		m.SetReplyTo(mail.NewEmail("", msg.ReplyTo))
	}

	if msg.Body.PlainText != "" {
		m.AddContent(mail.NewContent("text/plain", msg.Body.PlainText))
	}
	if msg.Body.HTML != "" {
		m.AddContent(mail.NewContent("text/html", msg.Body.HTML))
	}

	for _, att := range msg.Attachments {
		a := mail.NewAttachment()
		a.SetContent(base64.StdEncoding.EncodeToString(att.Content))
		a.SetFilename(att.Filename)
		if att.ContentType != "" {
			a.SetType(att.ContentType)
		}
		if att.Inline {
			a.SetDisposition("inline")
		} else {
			a.SetDisposition("attachment")
		}
		if att.ContentID != "" {
			a.SetContentID(att.ContentID)
		}
		m.AddAttachment(a)
	}

	resp, err := s.client.SendWithContext(ctx, m)
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
