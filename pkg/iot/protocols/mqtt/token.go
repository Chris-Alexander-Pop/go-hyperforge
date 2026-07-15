package mqtt

import (
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	paho "github.com/eclipse/paho.mqtt.golang"
)

// waitToken waits for an MQTT token with a timeout.
//
// paho.Token.WaitTimeout returns false when the timeout expires before completion.
// The previous pattern `if token.WaitTimeout(t) && token.Error() != nil` treated
// timeouts as success because a false WaitTimeout short-circuited the error check.
func waitToken(token paho.Token, timeout time.Duration, operation string) error {
	if token == nil {
		return pkgerrors.Internal("MQTT "+operation+": nil token", nil)
	}
	if !token.WaitTimeout(timeout) {
		return pkgerrors.DeadlineExceeded("MQTT "+operation+" timed out", nil)
	}
	if err := token.Error(); err != nil {
		return pkgerrors.Internal("failed to "+operation, err)
	}
	return nil
}
