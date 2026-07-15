package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type EmailTestSuite struct {
	test.Suite
	sender email.Sender
}

func (s *EmailTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.sender = memory.New()
}

func (s *EmailTestSuite) TestSend() {
	msg := &email.Message{
		To:      []string{"test@example.com"},
		Subject: "Test Subject",
		Body:    email.Body{PlainText: "Hello"},
	}
	err := s.sender.Send(s.Ctx, msg)
	s.NoError(err)
}

func (s *EmailTestSuite) TestSendBatch() {
	msgs := []*email.Message{
		{To: []string{"test1@example.com"}, Subject: "S1"},
		{To: []string{"test2@example.com"}, Subject: "S2"},
	}
	err := s.sender.SendBatch(s.Ctx, msgs)
	s.NoError(err)
}

func TestEmailSuite(t *testing.T) {
	test.Run(t, new(EmailTestSuite))
}
