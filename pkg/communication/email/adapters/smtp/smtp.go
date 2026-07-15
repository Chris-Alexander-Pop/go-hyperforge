package smtp

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

// Sender implements email.Sender for SMTP.
type Sender struct {
	host        string
	port        string
	username    string
	password    string
	defaultFrom string
}

// New creates a new SMTP sender.
func New(cfg email.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}
	if cfg.SMTPHost == "" {
		return nil, errors.InvalidArgument("SMTP host is required", nil)
	}

	return &Sender{
		host:        cfg.SMTPHost,
		port:        fmt.Sprintf("%d", cfg.SMTPPort),
		username:    cfg.SMTPUsername,
		password:    cfg.SMTPPassword,
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

	from := msg.From
	if from == "" {
		from = s.defaultFrom
	}
	if from == "" {
		return errors.InvalidArgument("from address is required", nil)
	}

	// Clone message with resolved From for MIME building.
	out := *msg
	out.From = from

	raw, err := email.BuildMIME(&out)
	if err != nil {
		return errors.Internal("failed to build MIME message", err)
	}

	addr := net.JoinHostPort(s.host, s.port)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	// Dial with context deadline when present; net/smtp.SendMail ignores ctx.
	if deadline, ok := ctx.Deadline(); ok {
		d := time.Until(deadline)
		if d <= 0 {
			return context.DeadlineExceeded
		}
		conn, err := net.DialTimeout("tcp", addr, d)
		if err != nil {
			return errors.Internal("failed to dial smtp", err)
		}
		defer conn.Close()
		_ = conn.SetDeadline(deadline)

		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return errors.Internal("failed to create smtp client", err)
		}
		defer client.Close()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return errors.Internal("smtp auth failed", err)
			}
		}
		if err := client.Mail(from); err != nil {
			return errors.Internal("smtp MAIL failed", err)
		}
		recipients := append(append([]string{}, msg.To...), msg.CC...)
		recipients = append(recipients, msg.BCC...)
		for _, rcpt := range recipients {
			if err := client.Rcpt(rcpt); err != nil {
				return errors.Internal("smtp RCPT failed", err)
			}
		}
		w, err := client.Data()
		if err != nil {
			return errors.Internal("smtp DATA failed", err)
		}
		if _, err := w.Write(raw); err != nil {
			_ = w.Close()
			return errors.Internal("smtp write failed", err)
		}
		if err := w.Close(); err != nil {
			return errors.Internal("smtp close failed", err)
		}
		return client.Quit()
	}

	recipients := append(append([]string{}, msg.To...), msg.CC...)
	recipients = append(recipients, msg.BCC...)
	if err := smtp.SendMail(addr, auth, from, recipients, raw); err != nil {
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
