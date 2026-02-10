package smtp

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

// Sender implements email.Sender for SMTP.
type Sender struct {
	host     string
	port     string
	username string
	password string
}

// New creates a new SMTP sender.
func New(cfg email.Config) (email.Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	return &Sender{
		host:     cfg.SMTPHost,
		port:     fmt.Sprintf("%d", cfg.SMTPPort),
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
	}, nil
}

// Send implements email.Sender.
func (s *Sender) Send(ctx context.Context, msg *email.Message) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	to := msg.To
	// Simple body construction. In a real world app this should likely use a library to handle MIME.
	body := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to[0], msg.Subject, msg.Body.PlainText)

	err := smtp.SendMail(addr, auth, msg.From, to, []byte(body))
	if err != nil {
		return errors.Internal("failed to send email via smtp", err)
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
