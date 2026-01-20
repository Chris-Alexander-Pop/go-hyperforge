package apns

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/push"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

// Sender implements push.Sender for APNS.
type Sender struct {
	client *apns2.Client
	topic  string
}

// New creates a new APNS sender.
func New(cfg push.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.APNSKeyFile == "" || cfg.APNSKeyID == "" || cfg.APNSTeamID == "" {
		return nil, errors.InvalidArgument("APNS credentials are required", nil)
	}

	authKey, err := token.AuthKeyFromFile(cfg.APNSKeyFile)
	if err != nil {
		return nil, errors.Internal("failed to load APNS key file", err)
	}

	token := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.APNSKeyID,
		TeamID:  cfg.APNSTeamID,
	}

	var client *apns2.Client
	if cfg.APNSDevelopment {
		client = apns2.NewTokenClient(token).Development()
	} else {
		client = apns2.NewTokenClient(token).Production()
	}

	return &Sender{
		client: client,
		topic:  cfg.APNSTopic,
	}, nil
}

// Send implements push.Sender.
func (s *Sender) Send(ctx context.Context, msg *push.Message) error {
	for _, deviceToken := range msg.Tokens {
		notification := &apns2.Notification{
			DeviceToken: deviceToken,
			Topic:       s.topic,
			Payload: map[string]interface{}{
				"aps": map[string]interface{}{
					"alert": map[string]interface{}{
						"title": msg.Title,
						"body":  msg.Body,
					},
					"sound": "default",
				},
				"data": msg.Data,
			},
		}

		if msg.Priority == "high" {
			notification.Priority = apns2.PriorityHigh
		} else if msg.Priority == "low" {
			notification.Priority = apns2.PriorityLow
		}

		// Set Context
		// apns2 library PushWithContext method exists? Or just Push.
		// checking lib docs (memory): client.Push(n) is standard. client.PushWithContext(ctx, n) might exist.
		// If not, we just use Push. The sideshow/apns2 library does support context in newer versions.

		res, err := s.client.PushWithContext(ctx, notification)
		if err != nil {
			return errors.Internal("failed to send apns notification", err)
		}
		if res.StatusCode != http.StatusOK {
			return errors.Internal(fmt.Sprintf("apns error: %s", res.Reason), nil)
		}
	}
	return nil
}

// SendBatch implements push.Sender.
func (s *Sender) SendBatch(ctx context.Context, msgs []*push.Message) error {
	for _, msg := range msgs {
		if err := s.Send(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

// Close implements push.Sender.
func (s *Sender) Close() error {
	return nil
}
