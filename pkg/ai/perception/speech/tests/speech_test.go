package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/speech/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type SpeechTestSuite struct {
	test.Suite
	client speech.SpeechClient
}

func (s *SpeechTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.client = memory.New()
}

func (s *SpeechTestSuite) TestSpeechToText() {
	text, err := s.client.SpeechToText(s.Ctx, []byte("audio-data"))
	s.NoError(err)
	s.Contains(text, "mock transcript")
}

func (s *SpeechTestSuite) TestTextToSpeech() {
	audio, err := s.client.TextToSpeech(s.Ctx, "Hello world", speech.FormatMP3)
	s.NoError(err)
	s.NotEmpty(audio)
}

func TestSpeechSuite(t *testing.T) {
	test.Run(t, new(SpeechTestSuite))
}
