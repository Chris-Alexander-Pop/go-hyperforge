package grpc_test

import (
	"context"
	"testing"

	apigrpc "github.com/chris-alexander-pop/go-hyperforge/pkg/api/grpc"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
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

type stubVerifier struct {
	sub   string
	roles []string
	err   error
}

func (v stubVerifier) Verify(ctx context.Context, token string) (string, []string, error) {
	if v.err != nil {
		return "", nil, v.err
	}
	if token != "good" {
		return "", nil, errors.Unauthorized("bad token", nil)
	}
	return v.sub, v.roles, nil
}

func TestAuthInterceptor(t *testing.T) {
	interceptor := apigrpc.AuthInterceptor(stubVerifier{sub: "alice", roles: []string{"admin"}})

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, nil
		})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	md := metadata.Pairs("authorization", "Bearer good")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test"},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			assert.Equal(t, "alice", apigrpc.GetSubject(ctx))
			assert.Equal(t, []string{"admin"}, apigrpc.GetRoles(ctx))
			return "ok", nil
		})
	require.NoError(t, err)
}

func TestStreamErrorInterceptor_MapsAppError(t *testing.T) {
	interceptor := apigrpc.StreamErrorInterceptor()
	err := interceptor(nil, &fakeStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/s"},
		func(srv interface{}, ss grpc.ServerStream) error {
			return errors.NotFound("missing", nil)
		})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeStream) Context() context.Context { return f.ctx }
