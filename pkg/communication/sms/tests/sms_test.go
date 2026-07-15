package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type SMSTestSuite struct {
	test.Suite
	sender sms.Sender
}

func (s *SMSTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.sender = memory.New()
}

func (s *SMSTestSuite) TestSend() {
	msg := sms.Message{
		To:   "+1234567890",
		Body: "Hello SMS",
	}
	err := s.sender.Send(s.Ctx, &msg)
	s.NoError(err)
}

func TestSMSSuite(t *testing.T) {
	test.Run(t, new(SMSTestSuite))
}
