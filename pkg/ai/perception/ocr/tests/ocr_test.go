package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/ocr"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/ocr/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type OCRTestSuite struct {
	test.Suite
	client ocr.OCRClient
}

func (s *OCRTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.client = memory.New()
}

func (s *OCRTestSuite) TestDetectText() {
	doc := ocr.Document{
		URI: "s3://bucket/doc.pdf",
	}

	result, err := s.client.DetectText(s.Ctx, doc)
	s.NoError(err)
	s.NotNil(result)
	s.Contains(result.Text, "mock OCR")
	s.NotEmpty(result.Pages)
}

func (s *OCRTestSuite) TestDetectText_Invalid() {
	doc := ocr.Document{} // Empty
	result, err := s.client.DetectText(s.Ctx, doc)
	s.Error(err)
	s.Nil(result)
}

func TestOCRSuite(t *testing.T) {
	test.Run(t, new(OCRTestSuite))
}
