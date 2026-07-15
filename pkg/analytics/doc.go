/*
Package analytics provides approximate uniqueness / cardinality tracking and
event ingest (memory + warehouse sinks).

# Scope

Tracker estimates how many distinct elements have been seen for a named
counter (e.g. unique visitors) via HyperLogLog. Sink ingests structured events
into an in-memory or warehouse-backed sink. WindowedUniqueness buckets
Tracker counters by time window.

It is intentionally not a full analytics product:

  - No sessionization, funnels, retention, or attribution
  - No OLAP dashboards (use pkg/data/bigdata Client for SQL)
  - Exact counters: use CounterStore (memory ExactStore) rather than HLL Tracker

# Adapters

  - adapters/memory — HLL Tracker + event Sink + ExactStore (CounterStore)
  - adapters/redis  — Redis native HyperLogLog (PFADD / PFCOUNT / PFMERGE)
  - adapters/warehouse — Sink writing INSERT rows via pkg/data/bigdata.Client

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
