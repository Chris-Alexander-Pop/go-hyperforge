package api_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/api"
	"github.com/chris-alexander-pop/system-design-library/pkg/api/rest"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := api.Config{
		Protocol: api.ProtocolREST,
		Port:     "8081",
	}

	server, err := api.New(cfg)
	require.NoError(t, err)

	if _, ok := server.(*rest.Server); !ok {
		t.Errorf("Expected *rest.Server, got %T", server)
	}

	_, err = api.New(api.Config{Protocol: "unknown"})
	assert.Error(t, err)
	assert.True(t, errors.IsCode(err, errors.CodeInvalidArgument))
}

func TestNewGraphQLRequiresSchema(t *testing.T) {
	_, err := api.New(api.Config{Protocol: api.ProtocolGraphQL, Port: "8080"})
	require.Error(t, err)
	assert.True(t, errors.IsCode(err, errors.CodeInvalidArgument))
}

func TestNewGRPC(t *testing.T) {
	server, err := api.New(api.Config{Protocol: api.ProtocolGRPC, Port: "9091"})
	require.NoError(t, err)
	require.NotNil(t, server)
}
