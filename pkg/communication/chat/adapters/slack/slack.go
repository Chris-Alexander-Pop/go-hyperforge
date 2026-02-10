package slack

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/chat"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/slack-go/slack"
)

// Sender implements chat.Sender for Slack.
type Sender struct {
	client *slack.Client
}

// New creates a new Slack sender.
func New(cfg chat.Config) (chat.Sender, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.SlackToken == "" {
		return nil, errors.InvalidArgument("Slack token is required", nil)
	}

	return &Sender{
		client: slack.New(cfg.SlackToken),
	}, nil
}

// Send implements chat.Sender.
func (s *Sender) Send(ctx context.Context, msg *chat.Message) error {
	channelID := msg.ChannelID
	if channelID == "" {
		channelID = msg.UserID // Fallback to UserID if ChannelID is empty, though Slack API treats them similarly often.
	}
	if channelID == "" {
		return errors.InvalidArgument("ChannelID or UserID required", nil)
	}

	options := []slack.MsgOption{
		slack.MsgOptionText(msg.Text, false),
	}

	if len(msg.Attachments) > 0 {
		var slackAttachments []slack.Attachment
		for _, att := range msg.Attachments {
			sa := slack.Attachment{
				Title:    att.Title,
				Text:     att.Text,
				ImageURL: att.ImageURL,
				Color:    att.Color,
			}
			for _, f := range att.Fields {
				sa.Fields = append(sa.Fields, slack.AttachmentField{
					Title: f.Title,
					Value: f.Value,
					Short: f.Short,
				})
			}
			slackAttachments = append(slackAttachments, sa)
		}
		options = append(options, slack.MsgOptionAttachments(slackAttachments...))
	}

	if msg.ThreadID != "" {
		options = append(options, slack.MsgOptionTS(msg.ThreadID))
	}

	// PostMessageContext is supported by newer versions of slack-go.
	// If not, we use PostMessage. slack-go/slack v0.12+ supports Context.
	_, _, err := s.client.PostMessageContext(ctx, channelID, options...)
	if err != nil {
		return errors.Internal("failed to send slack message", err)
	}

	return nil
}

// Close implements chat.Sender.
func (s *Sender) Close() error {
	return nil
}
