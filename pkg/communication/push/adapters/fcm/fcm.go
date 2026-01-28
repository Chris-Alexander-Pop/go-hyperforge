package fcm

import (
	"context"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/push"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Sender implements push.Sender for FCM.
type Sender struct {
	client *messaging.Client
}

// New creates a new FCM sender.
func New(ctx context.Context, cfg push.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	opts := []option.ClientOption{}
	if cfg.FCMServiceAccount != "" {
		// Use default scopes for FCM/Firebase
		creds, err := google.CredentialsFromJSON(ctx, []byte(cfg.FCMServiceAccount), "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, errors.Internal("failed to parse fcm credentials", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	app, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return nil, errors.Internal("failed to initialize firebase app", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, errors.Internal("failed to create fcm client", err)
	}

	return &Sender{
		client: client,
	}, nil
}

// Send implements push.Sender.
func (s *Sender) Send(ctx context.Context, msg *push.Message) error {
	// FCM supports sending to multiple tokens via MulticastMessage, but Sender.Send takes one message struct which has []Tokens.
	// If multiple tokens are present, we use Multicast.

	if len(msg.Tokens) == 0 {
		return errors.InvalidArgument("no tokens provided", nil)
	}

	fcmMsg := &messaging.MulticastMessage{
		Tokens: msg.Tokens,
		Notification: &messaging.Notification{
			Title:    msg.Title,
			Body:     msg.Body,
			ImageURL: msg.ImageURL,
		},
		Data: msg.Data,
	}

	if msg.Platform == "android" {
		fcmMsg.Android = &messaging.AndroidConfig{
			Priority: msg.Priority,
			TTL:      &msg.TTL,
		}
	} else if msg.Platform == "ios" {
		fcmMsg.APNS = &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Badge: func(i int) *int { return &i }(1), // Example
				},
			},
		}
	}

	br, err := s.client.SendEachForMulticast(ctx, fcmMsg)
	if err != nil {
		return errors.Internal("failed to send fcm multicast", err)
	}

	if br.FailureCount > 0 {
		// We could inspect br.Responses to find which tokens failed.
		// For now, if any failed, we log or return error.
		// Ideally we should return partial error or list of failed tokens, but interface returns single error.
		return errors.Internal("some fcm messages failed to send", nil)
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
