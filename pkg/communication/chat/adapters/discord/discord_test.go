package discord_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/chat"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/chat/adapters/discord"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresToken(t *testing.T) {
	_, err := discord.New(chat.Config{Driver: communication.DriverDiscord})
	require.Error(t, err)
}

func TestSendCanceledContext(t *testing.T) {
	s, err := discord.New(chat.Config{
		Driver:       communication.DriverDiscord,
		DiscordToken: "test-token",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.Send(ctx, &chat.Message{ChannelID: "1", Text: "hi"})
	require.ErrorIs(t, err, context.Canceled)
}
