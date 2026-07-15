package memory

import (
	"context"
	"sync/atomic"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/streaming"
)

// Ensure Client implements streaming.Client.
var _ streaming.Client = (*Client)(nil)

// Record represents a stored streaming record.
type Record struct {
	StreamName   string
	PartitionKey string
	Data         []byte
}

// Client implements streaming.Client in memory.
// Config.BufferSize caps retained records; <= 0 means unlimited.
type Client struct {
	mu         *concurrency.SmartMutex
	records    []Record
	bufferSize int
	closed     atomic.Bool
}

// New creates a new in-memory streaming client.
func New(cfg streaming.Config) *Client {
	return &Client{
		mu:         concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "StreamingMemory"}),
		records:    make([]Record, 0),
		bufferSize: cfg.BufferSize,
	}
}

func (c *Client) guard() error {
	if c.closed.Load() {
		return streaming.ErrClosed
	}
	return nil
}

func (c *Client) PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error {
	if err := c.guard(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.bufferSize > 0 && len(c.records) >= c.bufferSize {
		return streaming.ErrBufferFull
	}

	// Clone data to avoid race conditions if caller modifies it
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	c.records = append(c.records, Record{
		StreamName:   streamName,
		PartitionKey: partitionKey,
		Data:         dataCopy,
	})
	return nil
}

func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = nil
	return nil
}

// GetRecords is a test helper to inspect sent records.
func (c *Client) GetRecords() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.records == nil {
		return nil
	}
	out := make([]Record, len(c.records))
	copy(out, c.records)
	return out
}
