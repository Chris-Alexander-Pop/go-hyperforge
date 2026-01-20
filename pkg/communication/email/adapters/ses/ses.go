package ses

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

// Sender implements email.Sender for AWS SES.
type Sender struct {
	client *sesv2.Client
}

// New creates a new SES sender.
func New(ctx context.Context, cfg email.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.SESRegion))
	if err != nil {
		return nil, errors.Internal("failed to load aws config", err)
	}

	return &Sender{
		client: sesv2.NewFromConfig(awsCfg),
	}, nil
}

// Send implements email.Sender.
func (s *Sender) Send(ctx context.Context, msg *email.Message) error {
	input := &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data: aws.String(msg.Subject),
				},
				Body: &types.Body{},
			},
		},
		Destination: &types.Destination{},
	}

	if msg.Body.PlainText != "" {
		input.Content.Simple.Body.Text = &types.Content{
			Data: aws.String(msg.Body.PlainText),
		}
	}
	if msg.Body.HTML != "" {
		input.Content.Simple.Body.Html = &types.Content{
			Data: aws.String(msg.Body.HTML),
		}
	}

	if msg.From != "" {
		input.FromEmailAddress = aws.String(msg.From)
	}

	for _, to := range msg.To {
		input.Destination.ToAddresses = append(input.Destination.ToAddresses, to)
	}
	for _, cc := range msg.CC {
		input.Destination.CcAddresses = append(input.Destination.CcAddresses, cc)
	}
	for _, bcc := range msg.BCC {
		input.Destination.BccAddresses = append(input.Destination.BccAddresses, bcc)
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
