package messaging

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/bloomfilter"
)

// DeduplicatingConsumer wraps a Consumer with message deduplication using a Bloom filter.
// This prevents processing the same message multiple times in at-least-once delivery systems.
//
// Note: Due to Bloom filter properties, there's a small chance of false positives
// (skipping a message we haven't seen). Set the false positive rate appropriately.
type DeduplicatingConsumer struct {
	consumer Consumer
	bloom    *bloomfilter.BloomFilter
	inFlight map[string]struct{}
	mu       *concurrency.SmartRWMutex
}

// DeduplicationConfig configures the deduplication filter.
type DeduplicationConfig struct {
	// ExpectedMessages is the estimated number of unique messages to track.
	ExpectedMessages uint `env:"MSG_DEDUP_ELEMENTS" env-default:"1000000"`

	// FalsePositiveRate is the acceptable false positive rate.
	// Lower = more memory but fewer false skips.
	FalsePositiveRate float64 `env:"MSG_DEDUP_FPR" env-default:"0.001"`
}

// NewDeduplicatingConsumer wraps a consumer with Bloom filter deduplication.
func NewDeduplicatingConsumer(consumer Consumer, cfg DeduplicationConfig) *DeduplicatingConsumer {
	if cfg.ExpectedMessages == 0 {
		cfg.ExpectedMessages = 1000000
	}
	if cfg.FalsePositiveRate <= 0 {
		cfg.FalsePositiveRate = 0.001
	}
	return &DeduplicatingConsumer{
		consumer: consumer,
		bloom:    bloomfilter.New(cfg.ExpectedMessages, cfg.FalsePositiveRate),
		inFlight: make(map[string]struct{}),
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "DeduplicatingConsumer"}),
	}
}

func (dc *DeduplicatingConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	return dc.consumer.Consume(ctx, func(ctx context.Context, msg *Message) error {
		dedupKey := dc.getDeduplicationKey(msg)

		// Claim under write lock to close the check-then-act TOCTOU window
		// between concurrent handlers seeing the same unseen key.
		dc.mu.Lock()
		if dc.bloom.ContainsString(dedupKey) {
			dc.mu.Unlock()
			return nil
		}
		if _, busy := dc.inFlight[dedupKey]; busy {
			dc.mu.Unlock()
			return nil
		}
		dc.inFlight[dedupKey] = struct{}{}
		dc.mu.Unlock()

		err := handler(ctx, msg)

		dc.mu.Lock()
		delete(dc.inFlight, dedupKey)
		if err == nil {
			dc.bloom.AddString(dedupKey)
		}
		dc.mu.Unlock()

		return err
	})
}

func (dc *DeduplicatingConsumer) Close() error {
	return dc.consumer.Close()
}

func (dc *DeduplicatingConsumer) getDeduplicationKey(msg *Message) string {
	if msg.ID != "" {
		return msg.ID
	}
	return msg.Topic + ":" + string(msg.Payload)
}

// Stats returns deduplication statistics.
func (dc *DeduplicatingConsumer) Stats() DeduplicationStats {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return DeduplicationStats{
		TrackedMessages:   dc.bloom.Count(),
		FalsePositiveRate: dc.bloom.EstimatedFalsePositiveRate(),
	}
}

// DeduplicationStats contains deduplication statistics.
type DeduplicationStats struct {
	TrackedMessages   uint64
	FalsePositiveRate float64
}

// Reset clears the deduplication filter.
// Use with caution - messages seen before reset may be reprocessed.
func (dc *DeduplicatingConsumer) Reset() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.bloom.Clear()
	dc.inFlight = make(map[string]struct{})
}

var _ Consumer = (*DeduplicatingConsumer)(nil)
