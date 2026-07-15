package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type ScanningTestSuite struct {
	test.Suite
	scanner scanning.Scanner
}

func (s *ScanningTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.scanner = memory.New()
}

func (s *ScanningTestSuite) TestScan() {
	res := scanning.Resource{
		ID:       "file-1",
		Location: "/tmp/file",
	}

	report, err := s.scanner.Scan(s.Ctx, res)
	s.NoError(err)
	s.NotNil(report)
	s.True(report.Clean)
}

func TestScanningSuite(t *testing.T) {
	test.Run(t, new(ScanningTestSuite))
}
