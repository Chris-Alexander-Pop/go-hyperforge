package coap_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/protocols/coap"
	"github.com/stretchr/testify/require"
)

func TestUDPExchange(t *testing.T) {
	srv := coap.NewUDP(coap.Config{Timeout: 2 * time.Second})
	srv.RegisterHandler("/sensors/temp", func(ctx context.Context, req *coap.Message) (*coap.Message, error) {
		return &coap.Message{Code: coap.CodeContent, Payload: []byte("22.1")}, nil
	})
	srv.RegisterHandler("/echo", func(ctx context.Context, req *coap.Message) (*coap.Message, error) {
		return &coap.Message{Code: coap.CodeChanged, Payload: req.Payload}, nil
	})
	ctx := context.Background()
	require.NoError(t, srv.Listen(ctx, "127.0.0.1:0"))
	t.Cleanup(func() { _ = srv.Close() })

	addr := srv.LocalAddr().String()
	cli := coap.NewUDP(coap.Config{Address: addr, Timeout: 2 * time.Second})
	require.NoError(t, cli.Connect(ctx))
	t.Cleanup(func() { _ = cli.Close() })

	resp, err := cli.Get(ctx, "/sensors/temp")
	require.NoError(t, err)
	require.Equal(t, coap.CodeContent, resp.Code)
	require.Equal(t, "22.1", string(resp.Payload))

	resp, err = cli.Post(ctx, "/echo", []byte("ping"))
	require.NoError(t, err)
	require.Equal(t, coap.CodeChanged, resp.Code)
	require.Equal(t, "ping", string(resp.Payload))

	resp, err = cli.Get(ctx, "/missing")
	require.NoError(t, err)
	require.Equal(t, coap.CodeNotFound, resp.Code)
}

func TestUDPNotConnected(t *testing.T) {
	cli := coap.NewUDP(coap.Config{Address: "127.0.0.1:1"})
	_, err := cli.Get(context.Background(), "/x")
	require.Error(t, err)
}

func TestUDPConnectRequiresAddress(t *testing.T) {
	cli := coap.NewUDP(coap.Config{})
	err := cli.Connect(context.Background())
	require.Error(t, err)
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	// Exercise via UDP exchange with path segments and query.
	srv := coap.NewUDP(coap.Config{Timeout: time.Second})
	srv.RegisterHandler("/a/b", func(ctx context.Context, req *coap.Message) (*coap.Message, error) {
		return &coap.Message{Code: coap.CodeContent, Payload: []byte(req.Query)}, nil
	})
	require.NoError(t, srv.Listen(context.Background(), "127.0.0.1:0"))
	t.Cleanup(func() { _ = srv.Close() })

	cli := coap.NewUDP(coap.Config{Address: srv.LocalAddr().String(), Timeout: time.Second})
	require.NoError(t, cli.Connect(context.Background()))
	t.Cleanup(func() { _ = cli.Close() })

	resp, err := cli.Do(context.Background(), coap.Request{
		Method: coap.MethodGET,
		Path:   "/a/b",
		Query:  "k=v",
		Type:   coap.TypeConfirmable,
	})
	require.NoError(t, err)
	require.Equal(t, "k=v", string(resp.Payload))
}
