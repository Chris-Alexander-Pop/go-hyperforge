package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type PushTestSuite struct {
	test.Suite
	sender push.Sender
}

func (s *PushTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.sender = memory.New()
}

func (s *PushTestSuite) TestSend() {
	msg := &push.Message{
		Tokens: []string{"token-123"},
		Title:  "Hello",
		Body:   "World",
	}
	err := s.sender.Send(s.Ctx, msg)
	s.NoError(err)
}

func TestPushSuite(t *testing.T) {
	test.Run(t, new(PushTestSuite))
}
