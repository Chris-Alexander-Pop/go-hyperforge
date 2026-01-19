package memory

import (
	"context"
	"sync"

	"github.com/chris-alexander-pop/system-design-library/pkg/streaming"
)

// Record represents a stored streaming record.
type Record struct {
	StreamName   string
	PartitionKey string
	Data         []byte
}

// Client implements streaming.Client in memory.
type Client struct {
	mu      sync.Mutex
	records []Record
}

// New creates a new in-memory streaming client.
func New(_ streaming.Config) *Client {
	return &Client{
		records: make([]Record, 0),
	}
}

func (c *Client) PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	return nil
}

// GetRecords is a test helper to inspect sent records.
func (c *Client) GetRecords() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return copy
	out := make([]Record, len(c.records))
	copy(out, c.records)
	return out
}
