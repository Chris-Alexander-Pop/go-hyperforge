package coap_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/protocols/coap"
	"github.com/stretchr/testify/require"
)

func TestMemoryCoAPGetPost(t *testing.T) {
	c := coap.NewMemory(coap.Config{})
	ctx := context.Background()
	require.NoError(t, c.Connect(ctx))
	t.Cleanup(func() { _ = c.Close() })

	c.RegisterHandler("/sensors/temp", func(ctx context.Context, req *coap.Message) (*coap.Message, error) {
		return &coap.Message{Code: coap.CodeContent, Payload: []byte("21.5")}, nil
	})
	c.RegisterHandler("/actuators/led", func(ctx context.Context, req *coap.Message) (*coap.Message, error) {
		return &coap.Message{Code: coap.CodeChanged, Payload: req.Payload}, nil
	})

	resp, err := c.Get(ctx, "/sensors/temp")
	require.NoError(t, err)
	require.Equal(t, coap.CodeContent, resp.Code)
	require.Equal(t, "21.5", string(resp.Payload))

	resp, err = c.Post(ctx, "/actuators/led", []byte("on"))
	require.NoError(t, err)
	require.Equal(t, coap.CodeChanged, resp.Code)
	require.Equal(t, "on", string(resp.Payload))

	resp, err = c.Get(ctx, "/missing")
	require.NoError(t, err)
	require.Equal(t, coap.CodeNotFound, resp.Code)
}

func TestMemoryCoAPNotConnected(t *testing.T) {
	c := coap.NewMemory(coap.Config{})
	_, err := c.Get(context.Background(), "/x")
	require.Error(t, err)
}

func TestMemoryCoAPObserveUnimplemented(t *testing.T) {
	c := coap.NewMemory(coap.Config{})
	require.NoError(t, c.Connect(context.Background()))
	err := c.Observe(context.Background(), "/x", nil)
	require.True(t, errors.IsCode(err, errors.CodeUnimplemented))
}
