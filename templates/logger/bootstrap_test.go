package loggerbootstrap_test

import (
	"testing"

	loggerbootstrap "github.com/chris-alexander-pop/go-hyperforge/templates/logger"
)

func TestBootstrap(t *testing.T) {
	stop := loggerbootstrap.Bootstrap(loggerbootstrap.Config{
		Level:  "ERROR",
		Format: "TEXT",
		Async:  false,
	})
	stop()
}
