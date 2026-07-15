package starter_test

import (
	"os"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/templates/service/starter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	_ = os.Unsetenv("SERVICE_NAME")
	_ = os.Unsetenv("HTTP_ADDR")
	cfg, err := starter.Load()
	require.NoError(t, err)
	assert.Equal(t, "hyperforge-service", cfg.ServiceName)
	assert.Equal(t, ":8080", cfg.HTTPAddr)
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("SERVICE_NAME", "search-api")
	t.Setenv("HTTP_ADDR", ":9090")
	cfg, err := starter.Load()
	require.NoError(t, err)
	assert.Equal(t, "search-api", cfg.ServiceName)
	assert.Equal(t, ":9090", cfg.HTTPAddr)
}
