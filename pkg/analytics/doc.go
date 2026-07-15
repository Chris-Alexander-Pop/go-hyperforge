/*
Package analytics provides approximate uniqueness / cardinality tracking.

# Scope

This package estimates how many distinct elements have been seen for a named
counter (e.g. unique visitors, unique search queries). It is built on
HyperLogLog (pkg/datastructures/hyperloglog for the memory adapter; Redis
PFADD/PFCOUNT/PFMERGE for the redis adapter).

It is intentionally not a general analytics or metrics warehouse:

  - No event ingest model, schemas, or streaming sinks
  - No sessionization, funnels, retention, or attribution
  - No time-series rollups, dashboards, or OLAP queries
  - No exact counters (use metering, telemetry, or a database for those)

For warehouse-style analytics see pkg/data/bigdata and related adapters.
For usage billing events see pkg/metering. For structured audit trails see
pkg/audit.

# Semantics

  - Count on a missing counter returns (0, nil). ErrCounterNotFound is reserved
    for operations that require an existing sketch (currently Merge on a missing
    source).
  - Precision is 4–16 (matching HyperLogLog). Default is 14 (~16KB / ~0.8% error
    for the in-memory adapter). Redis HyperLogLog ignores Precision.

# Adapters

  - adapters/memory — in-process HLL sketches (tests, single-node)
  - adapters/redis  — Redis native HyperLogLog (PFADD / PFCOUNT / PFMERGE)

# Example

	import (
		"github.com/chris-alexander-pop/system-design-library/pkg/analytics"
		"github.com/chris-alexander-pop/system-design-library/pkg/analytics/adapters/memory"
	)

	tracker, err := memory.New(analytics.DefaultConfig())
	if err != nil {
		return err
	}
	defer tracker.Close()

	_ = tracker.Add(ctx, "visitors", userID)
	count, err := tracker.Count(ctx, "visitors")
	_ = tracker.Merge(ctx, "week", "monday")
	_ = tracker.Reset(ctx, "visitors")
*/
package analytics
