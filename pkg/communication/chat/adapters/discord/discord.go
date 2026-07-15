package discord

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/chat"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

// Sender implements chat.Sender for Discord.
type Sender struct {
	session *discordgo.Session
}

// New creates a new Discord sender.
func New(cfg chat.Config) (*Sender, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.DiscordToken == "" {
		return nil, errors.InvalidArgument("Discord token is required", nil)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, errors.Internal("failed to create discord session", err)
	}

	return &Sender{
		session: dg,
	}, nil
}

// Send implements chat.Sender.
func (s *Sender) Send(ctx context.Context, msg *chat.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if msg == nil {
		return errors.InvalidArgument("message is required", nil)
	}

	channelID := msg.ChannelID
	if channelID == "" {
		return errors.InvalidArgument("ChannelID is required for Discord", nil)
	}

	opts := []discordgo.RequestOption{discordgo.WithContext(ctx)}

	if len(msg.Attachments) == 0 {
		_, err := s.session.ChannelMessageSend(channelID, msg.Text, opts...)
		if err != nil {
			return errors.Internal("failed to send discord message", err)
		}
		return nil
	}

	embed := &discordgo.MessageEmbed{
		Description: msg.Text,
	}

	att := msg.Attachments[0]
	if att.Title != "" {
		embed.Title = att.Title
	}
	if att.Text != "" {
		embed.Description = att.Text
	}
	if att.ImageURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: att.ImageURL}
	}
	for _, f := range att.Fields {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   f.Title,
			Value:  f.Value,
			Inline: f.Short,
		})
	}

	_, err := s.session.ChannelMessageSendEmbed(channelID, embed, opts...)
	if err != nil {
		return errors.Internal("failed to send discord embed", err)
	}

	return nil
}

// Close implements chat.Sender.
func (s *Sender) Close() error {
	return s.session.Close()
}
