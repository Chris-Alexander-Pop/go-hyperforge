package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/chat"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/chat/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type ChatTestSuite struct {
	test.Suite
	sender chat.Sender
}

func (s *ChatTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.sender = memory.New()
}

func (s *ChatTestSuite) TestSend() {
	msg := &chat.Message{
		ChannelID: "C12345",
		Text:      "Hello Chat",
	}
	err := s.sender.Send(s.Ctx, msg)
	s.NoError(err)
}

func TestChatSuite(t *testing.T) {
	test.Run(t, new(ChatTestSuite))
}
