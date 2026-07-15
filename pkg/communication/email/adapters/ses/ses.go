package ses

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

// Sender implements email.Sender for AWS SES.
type Sender struct {
	client      *sesv2.Client
	defaultFrom string
}

// New creates a new SES sender.
func New(ctx context.Context, cfg email.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.SESRegion))
	if err != nil {
		return nil, errors.Internal("failed to load aws config", err)
	}

	return &Sender{
		client:      sesv2.NewFromConfig(awsCfg),
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

	input := &sesv2.SendEmailInput{
		Destination: &types.Destination{},
	}
	if from != "" {
		input.FromEmailAddress = aws.String(from)
	}
	if len(msg.To) > 0 {
		input.Destination.ToAddresses = append(input.Destination.ToAddresses, msg.To...)
	}
	if len(msg.CC) > 0 {
		input.Destination.CcAddresses = append(input.Destination.CcAddresses, msg.CC...)
	}
	if len(msg.BCC) > 0 {
		input.Destination.BccAddresses = append(input.Destination.BccAddresses, msg.BCC...)
	}
	if msg.ReplyTo != "" {
		input.ReplyToAddresses = []string{msg.ReplyTo}
	}

	// Attachments require Raw content.
	if len(msg.Attachments) > 0 {
		out := *msg
		out.From = from
		raw, err := email.BuildMIME(&out)
		if err != nil {
			return errors.Internal("failed to build MIME message", err)
		}
		input.Content = &types.EmailContent{
			Raw: &types.RawMessage{Data: raw},
		}
	} else {
		input.Content = &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(msg.Subject)},
				Body:    &types.Body{},
			},
		}
		if msg.Body.PlainText != "" {
			input.Content.Simple.Body.Text = &types.Content{Data: aws.String(msg.Body.PlainText)}
		}
		if msg.Body.HTML != "" {
			input.Content.Simple.Body.Html = &types.Content{Data: aws.String(msg.Body.HTML)}
		}
	}

	_, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return errors.Internal("failed to send email via ses", err)
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
