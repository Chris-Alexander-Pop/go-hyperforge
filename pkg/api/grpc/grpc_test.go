package grpc_test

import (
	"context"
	"testing"

	apigrpc "github.com/chris-alexander-pop/system-design-library/pkg/api/grpc"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestErrorInterceptor_MapsAppError(t *testing.T) {
	interceptor := apigrpc.ErrorInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, errors.NotFound("missing resource", nil)
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestErrorInterceptor_PreservesGRPCStatus(t *testing.T) {
	interceptor := apigrpc.ErrorInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.FailedPrecondition, "precondition")
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestNewRegistersHealth(t *testing.T) {
	s := apigrpc.New(apigrpc.Config{Port: "0"})
	require.NotNil(t, s.Health())
	require.NotNil(t, s.GRPC())

	// Health server is usable
	s.Health().SetServingStatus("demo", grpc_health_v1.HealthCheckResponse_SERVING)
}
