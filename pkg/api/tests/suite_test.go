package api_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/rest"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// APISuite migrates root api factory smoke tests onto pkg/test.Suite.
type APISuite struct {
	test.Suite
}

func (s *APISuite) TestNewREST() {
	server, err := api.New(api.Config{
		Protocol: api.ProtocolREST,
		Port:     "8081",
	})
	s.NoError(err)
	_, ok := server.(*rest.Server)
	s.True(ok)
}

func (s *APISuite) TestNewUnknownProtocol() {
	_, err := api.New(api.Config{Protocol: "unknown"})
	s.Error(err)
	s.True(errors.IsCode(err, errors.CodeInvalidArgument))
}

func (s *APISuite) TestNewGraphQLRequiresSchema() {
	_, err := api.New(api.Config{Protocol: api.ProtocolGraphQL, Port: "8080"})
	s.Error(err)
	s.True(errors.IsCode(err, errors.CodeInvalidArgument))
}

func (s *APISuite) TestNewGRPC() {
	server, err := api.New(api.Config{Protocol: api.ProtocolGRPC, Port: "9091"})
	s.NoError(err)
	s.NotNil(server)
}

func (s *APISuite) TestLoadConfigDefaults() {
	cfg, err := api.LoadConfig()
	s.NoError(err)
	s.Equal(api.ProtocolREST, cfg.Protocol)
	s.NotEmpty(cfg.Port)
}

func TestAPISuite(t *testing.T) {
	test.Run(t, new(APISuite))
}
