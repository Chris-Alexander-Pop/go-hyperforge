package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/config"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type ConfigSuite struct {
	*test.Suite
}

type TestConfig struct {
	Param string `env:"TEST_PARAM" env-default:"default"`
	Num   int    `env:"TEST_NUM" env-default:"42"`
}

type RequiredConfig struct {
	Name string `env:"REQUIRED_NAME" validate:"required"`
}

type NestedConfig struct {
	Service string `env:"SERVICE_NAME" env-default:"api" validate:"required"`
	DB      NestedDBConfig
}

type NestedDBConfig struct {
	Host string `env:"DB_HOST" env-default:"localhost" validate:"required"`
	Port int    `env:"DB_PORT" env-default:"5432" validate:"gte=1"`
}

type CustomTagConfig struct {
	Slug     string `env:"CFG_SLUG" validate:"required,slug"`
	Phone    string `env:"CFG_PHONE" validate:"required,phone_e164"`
	Password string `env:"CFG_PASSWORD" validate:"required,password_strong"`
}

func TestConfigSuite(t *testing.T) {
	test.Run(t, &ConfigSuite{Suite: test.NewSuite()})
}

func (s *ConfigSuite) TestLoad_Defaults() {
	os.Unsetenv("TEST_PARAM")
	os.Unsetenv("TEST_NUM")

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

func (s *ConfigSuite) TestLoad_ValidationFailure() {
	os.Unsetenv("REQUIRED_NAME")

	var cfg RequiredConfig
	err := config.Load(&cfg)

	s.Error(err)
	s.True(pkgerrors.IsCode(err, pkgerrors.CodeInvalidArgument))
}

func (s *ConfigSuite) TestLoadFrom_EnvFile() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "app.env")
	content := "TEST_PARAM=from-file\nTEST_NUM=99\n"
	s.NoError(os.WriteFile(path, []byte(content), 0o600))

	os.Unsetenv("TEST_PARAM")
	os.Unsetenv("TEST_NUM")

	var cfg TestConfig
	err := config.LoadFrom(path, &cfg)

	s.NoError(err)
	s.Equal("from-file", cfg.Param)
	s.Equal(99, cfg.Num)
}

func (s *ConfigSuite) TestLoadFrom_MissingFile() {
	var cfg TestConfig
	err := config.LoadFrom(filepath.Join(s.T().TempDir(), "missing.env"), &cfg)

	s.Error(err)
	s.True(pkgerrors.IsCode(err, pkgerrors.CodeInternal))
}

func (s *ConfigSuite) TestLoadFrom_EmptyPath() {
	var cfg TestConfig
	err := config.LoadFrom("", &cfg)

	s.Error(err)
	s.True(pkgerrors.IsCode(err, pkgerrors.CodeInvalidArgument))
}

func (s *ConfigSuite) TestLoad_NestedStructs() {
	os.Setenv("SERVICE_NAME", "billing")
	os.Setenv("DB_HOST", "db.internal")
	os.Setenv("DB_PORT", "3306")
	defer func() {
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
	}()

	var cfg NestedConfig
	err := config.Load(&cfg)

	s.NoError(err)
	s.Equal("billing", cfg.Service)
	s.Equal("db.internal", cfg.DB.Host)
	s.Equal(3306, cfg.DB.Port)
}

func (s *ConfigSuite) TestLoad_CustomValidatorTags() {
	os.Setenv("CFG_SLUG", "my-service")
	os.Setenv("CFG_PHONE", "+14155552671")
	os.Setenv("CFG_PASSWORD", "Str0ng!Pass")
	defer func() {
		os.Unsetenv("CFG_SLUG")
		os.Unsetenv("CFG_PHONE")
		os.Unsetenv("CFG_PASSWORD")
	}()

	var cfg CustomTagConfig
	err := config.Load(&cfg)

	s.NoError(err)
	s.Equal("my-service", cfg.Slug)
	s.Equal("+14155552671", cfg.Phone)
	s.Equal("Str0ng!Pass", cfg.Password)
}

func (s *ConfigSuite) TestLoad_CustomValidatorTags_Failure() {
	os.Setenv("CFG_SLUG", "Not A Slug")
	os.Setenv("CFG_PHONE", "555-1234")
	os.Setenv("CFG_PASSWORD", "weak")
	defer func() {
		os.Unsetenv("CFG_SLUG")
		os.Unsetenv("CFG_PHONE")
		os.Unsetenv("CFG_PASSWORD")
	}()

	var cfg CustomTagConfig
	err := config.Load(&cfg)

	s.Error(err)
	s.True(pkgerrors.IsCode(err, pkgerrors.CodeInvalidArgument))
}

func (s *ConfigSuite) TestLoad_DotEnvFile() {
	dir := s.T().TempDir()
	prev, err := os.Getwd()
	s.NoError(err)
	s.NoError(os.Chdir(dir))
	defer func() { _ = os.Chdir(prev) }()

	s.NoError(os.WriteFile(".env", []byte("TEST_PARAM=from-dotenv\nTEST_NUM=7\n"), 0o600))
	os.Unsetenv("TEST_PARAM")
	os.Unsetenv("TEST_NUM")

	var cfg TestConfig
	err = config.Load(&cfg)

	s.NoError(err)
	s.Equal("from-dotenv", cfg.Param)
	s.Equal(7, cfg.Num)
}
