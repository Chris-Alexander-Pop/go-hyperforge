package twilio

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Sender implements sms.Sender for Twilio.
type Sender struct {
	client *twilio.RestClient
	from   string
}

// New creates a new Twilio sender.
func New(cfg sms.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.TwilioAccountSID == "" || cfg.TwilioAuthToken == "" {
		return nil, errors.InvalidArgument("Twilio credentials are required", nil)
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.TwilioAccountSID,
		Password: cfg.TwilioAuthToken,
	})

	return &Sender{
		client: client,
		from:   cfg.TwilioFromNumber, // Optional default from
	}, nil
}

// Send implements sms.Sender.
func (s *Sender) Send(ctx context.Context, msg *sms.Message) error {
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(msg.To)

	if msg.From != "" {
		params.SetFrom(msg.From)
	} else if s.from != "" {
		params.SetFrom(s.from)
	} else {
		return errors.InvalidArgument("twilio requires a 'From' number in the message or config", nil)
	}

	params.SetBody(msg.Body)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return errors.Internal("failed to send sms via twilio", err)
	}

	return nil
}

// SendBatch implements sms.Sender.
func (s *Sender) SendBatch(ctx context.Context, msgs []*sms.Message) error {
	// Twilio supports Messaging Services for bulk but standard API is per-message.
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
