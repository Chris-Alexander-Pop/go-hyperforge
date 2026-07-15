package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/vision"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/perception/vision/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type VisionTestSuite struct {
	test.Suite
	client vision.ComputerVision
}

func (s *VisionTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.client = memory.New()
}

func (s *VisionTestSuite) TestAnalyzeImage() {
	img := vision.Image{URI: "s3://bucket/image.jpg"}
	analysis, err := s.client.AnalyzeImage(s.Ctx, img, []vision.Feature{vision.FeatureLabels})
	s.NoError(err)
	s.NotNil(analysis)
	s.NotEmpty(analysis.Labels)
}

func (s *VisionTestSuite) TestDetectFaces() {
	img := vision.Image{URI: "s3://bucket/faces.jpg"}
	faces, err := s.client.DetectFaces(s.Ctx, img)
	s.NoError(err)
	s.NotEmpty(faces)
}

func TestVisionSuite(t *testing.T) {
	test.Run(t, new(VisionTestSuite))
}
