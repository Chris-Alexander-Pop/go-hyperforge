package config_test

import (
	"os"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/config"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type ConfigSuite struct {
	*test.Suite
}

type TestConfig struct {
	Param string `env:"TEST_PARAM" env-default:"default"`
	Num   int    `env:"TEST_NUM" env-default:"42"`
}

func TestConfigSuite(t *testing.T) {
	test.Run(t, &ConfigSuite{Suite: test.NewSuite()})
}

func (s *ConfigSuite) TestLoad_Defaults() {
	os.Unsetenv("TEST_PARAM")

	var cfg TestConfig
	err := config.Load(&cfg)

	s.NoError(err)
	s.Equal("default", cfg.Param)
	s.Equal(42, cfg.Num)
}

func (s *ConfigSuite) TestLoad_EnvVar() {
	os.Setenv("TEST_PARAM", "custom suite output")
	defer os.Unsetenv("TEST_PARAM")

	var cfg TestConfig
	err := config.Load(&cfg)

	s.NoError(err)
	s.Equal("custom suite output", cfg.Param)
}
