package apns_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push/adapters/apns"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresCredentials(t *testing.T) {
	_, err := apns.New(push.Config{Driver: communication.DriverAPNS})
	require.Error(t, err)
}
