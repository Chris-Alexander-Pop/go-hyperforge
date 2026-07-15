// Package metering provides mechanisms for tracking resource consumption and calculating costs.
//
// It supports recording usage events and applying rate cards for billing.
//
// Adapters:
//   - adapters/memory — in-process Meter and Rater (the only driver today)
//
// Observability and events:
//   - NewInstrumentedMeter / NewInstrumentedRater wrap adapters with logging and tracing
//   - NewEventedMeter optionally publishes metering.usage.recorded via pkg/events after RecordUsage
//
// Basic usage:
//
//	import (
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/metering"
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/metering/adapters/memory"
//	)
//
//	m := memory.New()
//	defer m.Close()
//
//	meter := metering.NewInstrumentedMeter(m)
//	rater := metering.NewInstrumentedRater(m)
package metering
