package slack_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/chat"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/chat/adapters/slack"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresToken(t *testing.T) {
	_, err := slack.New(chat.Config{Driver: communication.DriverSlack})
	require.Error(t, err)
}

func TestSendCanceledContext(t *testing.T) {
	s, err := slack.New(chat.Config{
		Driver:     communication.DriverSlack,
		SlackToken: "xoxb-test",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.Send(ctx, &chat.Message{ChannelID: "C1", Text: "hi"})
	require.ErrorIs(t, err, context.Canceled)
}
