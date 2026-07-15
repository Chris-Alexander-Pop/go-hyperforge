// Package templates holds service starter scaffolds and logger bootstrap examples
// for Hyperforge (see templates/service/starter and templates/logger).
//
// Logger Init bootstrap (apps must call Init before logging):
//
//	import (
//		"context"
//
//		"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
//		loggerbootstrap "github.com/chris-alexander-pop/go-hyperforge/templates/logger"
//	)
//
//	func main() {
//		stop := loggerbootstrap.Bootstrap(loggerbootstrap.Config{Level: "INFO", Async: true})
//		defer stop()
//		// or: logger.Init(logger.Config{Level: "INFO"}); defer logger.Shutdown(context.Background())
//		logger.L().Info("service starting")
//	}
//
// Prefer templates/service/starter.Bootstrap for config.Load + logger.Init together.
package templates
