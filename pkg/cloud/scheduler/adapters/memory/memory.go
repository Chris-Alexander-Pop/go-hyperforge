package memory

import (
	"context"
	"math/rand"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/scheduler"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// hostRecord tracks capacity used for placement decisions.
type hostRecord struct {
	Host     cloud.Host
	Placed   int // number of successful placements (for spread)
	Reserved cloud.Resources
}

// MemoryScheduler implements binpack / spread / random placement over an in-memory host pool.
type MemoryScheduler struct {
	hosts    map[string]*hostRecord
	strategy string
	mu       *concurrency.SmartRWMutex
	rng      *rand.Rand
}

// New creates a MemoryScheduler with the given strategy (defaults to random).
func New(cfg ...scheduler.Config) *MemoryScheduler {
	strategy := scheduler.StrategyRandom
	if len(cfg) > 0 && cfg[0].Strategy != "" {
		strategy = cfg[0].Strategy
	}
	return &MemoryScheduler{
		hosts:    make(map[string]*hostRecord),
		strategy: strategy,
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-scheduler",
		}),
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// AddHost registers a host in the scheduler pool.
func (s *MemoryScheduler) AddHost(host cloud.Host) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := host
	if h.Available.VCPUs == 0 && h.Available.MemoryMB == 0 && h.Available.DiskGB == 0 {
		h.Available = h.Capacity
	}
	s.hosts[host.ID] = &hostRecord{Host: h}
}

// AddHostID is a convenience for tests that only care about identity (infinite capacity).
func (s *MemoryScheduler) AddHostID(hostID string) {
	s.AddHost(cloud.Host{
		ID:     hostID,
		Status: cloud.HostStatusReady,
		Capacity: cloud.Resources{
			VCPUs:    64,
			MemoryMB: 256000,
			DiskGB:   10000,
		},
		Available: cloud.Resources{
			VCPUs:    64,
			MemoryMB: 256000,
			DiskGB:   10000,
		},
	})
}

func (s *MemoryScheduler) SelectHost(ctx context.Context, req scheduler.Requirement) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidates := s.eligibleLocked(req)
	if len(candidates) == 0 {
		return "", scheduler.ErrNoHostFound
	}

	var chosen *hostRecord
	switch s.strategy {
	case scheduler.StrategyBinpack:
		chosen = selectBinpack(candidates, req.Resources)
	case scheduler.StrategySpread:
		chosen = selectSpread(candidates)
	default:
		chosen = candidates[s.rng.Intn(len(candidates))]
	}

	// Reserve resources so subsequent placements see updated availability.
	reserve(&chosen.Host.Available, req.Resources)
	chosen.Reserved.VCPUs += req.Resources.VCPUs
	chosen.Reserved.MemoryMB += req.Resources.MemoryMB
	chosen.Reserved.DiskGB += req.Resources.DiskGB
	chosen.Reserved.GPUs += req.Resources.GPUs
	chosen.Placed++

	return chosen.Host.ID, nil
}

func (s *MemoryScheduler) eligibleLocked(req scheduler.Requirement) []*hostRecord {
	out := make([]*hostRecord, 0, len(s.hosts))
	for _, rec := range s.hosts {
		h := rec.Host
		if h.Status != cloud.HostStatusReady && h.Status != cloud.HostStatusUnknown && h.Status != "" {
			continue
		}
		if req.Zone != "" && h.Zone != "" && h.Zone != req.Zone {
			continue
		}
		if !fits(h.Available, req.Resources) {
			continue
		}
		if len(req.Tags) > 0 && !tagsMatch(h.Tags, req.Tags) {
			continue
		}
		out = append(out, rec)
	}
	return out
}

func fits(avail, need cloud.Resources) bool {
	return avail.VCPUs >= need.VCPUs &&
		avail.MemoryMB >= need.MemoryMB &&
		avail.DiskGB >= need.DiskGB &&
		avail.GPUs >= need.GPUs
}

func reserve(avail *cloud.Resources, need cloud.Resources) {
	avail.VCPUs -= need.VCPUs
	avail.MemoryMB -= need.MemoryMB
	avail.DiskGB -= need.DiskGB
	avail.GPUs -= need.GPUs
}

func tagsMatch(hostTags, required map[string]string) bool {
	if hostTags == nil {
		return false
	}
	for k, v := range required {
		if hostTags[k] != v {
			return false
		}
	}
	return true
}

// selectBinpack prefers the host with the least remaining capacity that still fits
// (pack tightly / consolidate).
func selectBinpack(candidates []*hostRecord, need cloud.Resources) *hostRecord {
	best := candidates[0]
	bestScore := remainingScore(best.Host.Available)
	for _, c := range candidates[1:] {
		score := remainingScore(c.Host.Available)
		if score < bestScore {
			best = c
			bestScore = score
		}
	}
	_ = need
	return best
}

// selectSpread prefers the host with the fewest placements so far.
func selectSpread(candidates []*hostRecord) *hostRecord {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.Placed < best.Placed {
			best = c
		} else if c.Placed == best.Placed && remainingScore(c.Host.Available) > remainingScore(best.Host.Available) {
			// Tie-break: more free capacity.
			best = c
		}
	}
	return best
}

func remainingScore(r cloud.Resources) int {
	return r.VCPUs*1000 + r.MemoryMB + r.DiskGB*10 + r.GPUs*5000
}

// Ensure MemoryScheduler implements Scheduler.
var _ scheduler.Scheduler = (*MemoryScheduler)(nil)
