package twilio

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
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
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.TwilioAccountSID == "" || cfg.TwilioAuthToken == "" {
		return nil, errors.InvalidArgument("Twilio credentials are required", nil)
	}

	from := cfg.TwilioFromNumber
	if from == "" {
		from = cfg.DefaultFrom
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.TwilioAccountSID,
		Password: cfg.TwilioAuthToken,
	})

	return &Sender{
		client: client,
		from:   from,
	}, nil
}

// Send implements sms.Sender.
//
// Note: the Twilio Go SDK CreateMessage API does not accept context.Context;
// we honor cancellation/deadline before the request is issued.
func (s *Sender) Send(ctx context.Context, msg *sms.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if msg == nil {
		return errors.InvalidArgument("message is required", nil)
	}
	if msg.To == "" {
		return errors.InvalidArgument("recipient is required", nil)
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(msg.To)

	switch {
	case msg.From != "":
		params.SetFrom(msg.From)
	case s.from != "":
		params.SetFrom(s.from)
	default:
		return errors.InvalidArgument("twilio requires a 'From' number in the message or config", nil)
	}

	params.SetBody(msg.Body)
	if msg.MediaURL != "" {
		params.SetMediaUrl([]string{msg.MediaURL})
	}

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return errors.Internal("failed to send sms via twilio", err)
	}

	return nil
}

// SendBatch implements sms.Sender.
func (s *Sender) SendBatch(ctx context.Context, msgs []*sms.Message) error {
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
