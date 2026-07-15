package mqtt_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	iotmqtt "github.com/chris-alexander-pop/go-hyperforge/pkg/iot/adapters/mqtt"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresBroker(t *testing.T) {
	_, err := iotmqtt.New(iotmqtt.Config{})
	require.Error(t, err)
}

func TestNewFromClientNil(t *testing.T) {
	_, err := iotmqtt.NewFromClient(nil)
	require.Error(t, err)
}

func TestAdapterImplementsClient(t *testing.T) {
	// Compile-time + construction without dialing: empty broker fails New.
	// Verify interface assignment with a typed nil check pattern via NewFromClient error path.
	var _ iot.Client = (*iotmqtt.Adapter)(nil)
	_ = context.Background()
}
