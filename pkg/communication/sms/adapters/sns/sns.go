package sns

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

// Sender implements sms.Sender for AWS SNS.
type Sender struct {
	client *sns.Client
}

// New creates a new SNS sender.
func New(ctx context.Context, cfg sms.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.SNSRegion == "" {
		return nil, errors.InvalidArgument("SNS region is required", nil)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.SNSRegion))
	if err != nil {
		return nil, errors.Internal("failed to load aws config", err)
	}

	return &Sender{
		client: sns.NewFromConfig(awsCfg),
	}, nil
}

// Send implements sms.Sender.
func (s *Sender) Send(ctx context.Context, msg *sms.Message) error {
	input := &sns.PublishInput{
		Message:     aws.String(msg.Body),
		PhoneNumber: aws.String(msg.To),
	}

	_, err := s.client.Publish(ctx, input)
	if err != nil {
		return errors.Internal("failed to send sms via sns", err)
	}

	return nil
}

// SendBatch implements sms.Sender.
func (s *Sender) SendBatch(ctx context.Context, msgs []*sms.Message) error {
	// SNS PublishBatch API for Standard Topics exists, but for SMS (Phone Numbers), it's typically individual.
	// We'll iterate.
	for _, msg := range msgs {
		if err := s.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

// Close implements sms.Sender.
func (s *Sender) Close() error {
	return nil
}
