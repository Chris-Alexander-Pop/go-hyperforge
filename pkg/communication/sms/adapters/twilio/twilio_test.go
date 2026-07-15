package twilio_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms/adapters/twilio"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresCredentials(t *testing.T) {
	_, err := twilio.New(sms.Config{Driver: communication.DriverTwilio})
	require.Error(t, err)
}

func TestSendCanceledContext(t *testing.T) {
	s, err := twilio.New(sms.Config{
		Driver:           communication.DriverTwilio,
		TwilioAccountSID: "AC123",
		TwilioAuthToken:  "token",
		TwilioFromNumber: "+15551212",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = s.Send(ctx, &sms.Message{To: "+15550000", Body: "hi", MediaURL: "https://example.com/a.jpg"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSendRequiresFrom(t *testing.T) {
	s, err := twilio.New(sms.Config{
		Driver:           communication.DriverTwilio,
		TwilioAccountSID: "AC123",
		TwilioAuthToken:  "token",
	})
	require.NoError(t, err)
	err = s.Send(context.Background(), &sms.Message{To: "+15550000", Body: "hi"})
	require.Error(t, err)
}
