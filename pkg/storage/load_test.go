package storage_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/storage"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type LoadConfigSuite struct {
	test.Suite
}

func (s *LoadConfigSuite) TestLoadConfigDefaults() {
	cfg, err := storage.LoadConfig()
	s.NoError(err)
	s.Equal("local", cfg.BlobDriver)
	s.Equal("memory", cfg.FileDriver)
	s.Equal("memory", cfg.BlockDriver)
	s.Equal("memory", cfg.ArchiveDriver)
	s.Equal("memory", cfg.ControllerDriver)
}

func TestLoadConfigSuite(t *testing.T) {
	test.Run(t, new(LoadConfigSuite))
}
