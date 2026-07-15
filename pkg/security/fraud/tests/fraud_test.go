package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/fraud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/fraud/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type FraudTestSuite struct {
	test.Suite
	detector fraud.Detector
}

func (s *FraudTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.detector = memory.New()
}

func (s *FraudTestSuite) TestScore() {
	evt := fraud.UserEvent{
		UserID:    "user-123",
		IPAddress: "127.0.0.1",
		Action:    "login",
	}
	evaluation, err := s.detector.Score(s.Ctx, evt)
	s.NoError(err)
	s.NotNil(evaluation)
	// Expect safe score by default
	s.Equal("allow", evaluation.Action)
}

func TestFraudSuite(t *testing.T) {
	test.Run(t, new(FraudTestSuite))
}
