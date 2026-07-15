package compute_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/compute"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type LoadConfigSuite struct {
	test.Suite
}

func (s *LoadConfigSuite) TestLoadConfigDefaults() {
	cfg, err := compute.LoadConfig()
	s.NoError(err)
	s.Equal("memory", cfg.VMDriver)
	s.Equal("memory", cfg.ContainerDriver)
	s.Equal("memory", cfg.ServerlessDriver)
}

func TestLoadConfigSuite(t *testing.T) {
	test.Run(t, new(LoadConfigSuite))
}
