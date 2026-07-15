package memory

import (
	"context"
	"sync/atomic"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/streaming"
)

// Ensure Client implements streaming.Client and can produce a Consumer.
var (
	_ streaming.Client   = (*Client)(nil)
	_ streaming.Consumer = (*Consumer)(nil)
)

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

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	c.records = append(c.records, Record{
		StreamName:   streamName,
		PartitionKey: partitionKey,
		Data:         dataCopy,
	})
	return nil
}

func (c *Client) PutRecords(ctx context.Context, records []streaming.Record) error {
	if err := c.guard(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.bufferSize > 0 && len(c.records)+len(records) > c.bufferSize {
		return streaming.ErrBufferFull
	}

	for _, r := range records {
		dataCopy := make([]byte, len(r.Data))
		copy(dataCopy, r.Data)
		c.records = append(c.records, Record{
			StreamName:   r.StreamName,
			PartitionKey: r.PartitionKey,
			Data:         dataCopy,
		})
	}
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

// NewConsumer returns a Consumer that reads from this client's buffer.
func (c *Client) NewConsumer() *Consumer {
	return &Consumer{client: c}
}

// Consumer reads records from an in-memory Client.
type Consumer struct {
	client *Client
	offset int
	closed atomic.Bool
}

func (c *Consumer) Consume(ctx context.Context, streamName string, handler streaming.RecordHandler) error {
	if c.closed.Load() {
		return streaming.ErrClosed
	}
	if handler == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	records := c.client.GetRecords()
	for i := c.offset; i < len(records); i++ {
		if err := ctx.Err(); err != nil {
			c.offset = i
			return err
		}
		if c.closed.Load() {
			c.offset = i
			return streaming.ErrClosed
		}
		r := records[i]
		if streamName != "" && r.StreamName != streamName {
			continue
		}
		rec := streaming.Record{
			StreamName:   r.StreamName,
			PartitionKey: r.PartitionKey,
			Data:         append([]byte(nil), r.Data...),
		}
		if err := handler(ctx, rec); err != nil {
			c.offset = i
			return err
		}
	}
	c.offset = len(records)
	return nil
}

func (c *Consumer) Close() error {
	c.closed.Store(true)
	return nil
}
