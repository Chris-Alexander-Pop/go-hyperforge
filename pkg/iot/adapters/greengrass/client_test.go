package greengrass_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/adapters/greengrass"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePub struct {
	calls atomic.Int32
	last  string
}

func (f *fakePub) Publish(ctx context.Context, topic string, payload []byte) error {
	f.calls.Add(1)
	f.last = topic
	return nil
}

func TestAdapter_ImplementsClient(t *testing.T) {
	pub := &fakePub{}
	client, err := greengrass.NewAdapter(pub)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, client.Connect(ctx))
	assert.True(t, client.IsConnected())

	var got atomic.Pointer[iot.Message]
	require.NoError(t, client.Subscribe(ctx, "gg/+/telemetry", func(msg *iot.Message) {
		cp := *msg
		got.Store(&cp)
	}))

	require.NoError(t, client.Publish(ctx, "gg/core1/telemetry", []byte(`{"ok":1}`)))
	assert.Equal(t, int32(1), pub.calls.Load())
	msg := got.Load()
	require.NotNil(t, msg)
	assert.Equal(t, []byte(`{"ok":1}`), msg.Payload)

	client.Disconnect()
	assert.False(t, client.IsConnected())
}

func TestAdapter_NilPublisher(t *testing.T) {
	_, err := greengrass.NewAdapter(nil)
	require.Error(t, err)
}
