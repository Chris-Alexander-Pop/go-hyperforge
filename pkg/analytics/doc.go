/*
Package analytics provides approximate uniqueness / cardinality tracking and
lightweight event ingest for tests.

# Scope

Tracker estimates how many distinct elements have been seen for a named
counter (e.g. unique visitors) via HyperLogLog. Sink ingests structured events
into an in-memory (or future warehouse) sink. WindowedUniqueness buckets
Tracker counters by time window.

It is intentionally not a full analytics warehouse:

  - No sessionization, funnels, retention, or attribution
  - No OLAP queries or dashboards
  - Exact counters belong in metering / telemetry / database packages

# Adapters

  - adapters/memory — HLL Tracker + event Sink
  - adapters/redis  — Redis native HyperLogLog (PFADD / PFCOUNT / PFMERGE)

# Example

	tracker, _ := memory.New(analytics.DefaultConfig())
	defer tracker.Close()
	_ = tracker.Add(ctx, "visitors", userID)

	sink := memory.NewSink()
	_ = sink.Ingest(ctx, analytics.Event{Name: "page_view", UserID: userID})

	w := analytics.NewWindowedUniqueness(tracker, "visitors", time.Hour)
	_ = w.Add(ctx, userID, time.Now())
*/
package analytics
