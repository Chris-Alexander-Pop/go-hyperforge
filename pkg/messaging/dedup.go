package messaging

import (
	"context"
	"sync"

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
	mu       sync.RWMutex
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
	return &DeduplicatingConsumer{
		consumer: consumer,
		bloom:    bloomfilter.New(cfg.ExpectedMessages, cfg.FalsePositiveRate),
	}
}

func (dc *DeduplicatingConsumer) Consume(ctx context.Context, handler MessageHandler) error {
	return dc.consumer.Consume(ctx, func(ctx context.Context, msg *Message) error {
		// Generate deduplication key (use message ID or hash of payload)
		dedupKey := dc.getDeduplicationKey(msg)

		// Check if we've seen this message
		dc.mu.RLock()
		seen := dc.bloom.ContainsString(dedupKey)
		dc.mu.RUnlock()

		if seen {
			// Already processed (or false positive), skip
			return nil
		}

		// Process the message
		err := handler(ctx, msg)
		if err != nil {
			return err
		}

		// Mark as processed
		dc.mu.Lock()
		dc.bloom.AddString(dedupKey)
		dc.mu.Unlock()

		return nil
	})
}

func (dc *DeduplicatingConsumer) Close() error {
	return dc.consumer.Close()
}

func (dc *DeduplicatingConsumer) getDeduplicationKey(msg *Message) string {
	// Prefer message ID, fall back to topic+payload hash
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
}
