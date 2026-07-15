package streaming_test

import (
	"context"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/streaming"
	"github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"

	"github.com/stretchr/testify/suite"
)

type ClientSuite struct {
	*test.Suite
	newClient func() (streaming.Client, *memory.Client, error)
	name      string
}

func (s *ClientSuite) SetupTest() {
	s.Suite.SetupTest()
}

func (s *ClientSuite) TestPutRecordsAndConsume() {
	client := memory.New(streaming.Config{BufferSize: 100})
	defer client.Close()

	recs := []streaming.Record{
		{StreamName: "orders", PartitionKey: "u1", Data: []byte("a")},
		{StreamName: "orders", PartitionKey: "u2", Data: []byte("b")},
		{StreamName: "other", PartitionKey: "u3", Data: []byte("c")},
	}
	s.Require().NoError(client.PutRecords(s.Ctx, recs))
	s.Len(client.GetRecords(), 3)

	consumer := client.NewConsumer()
	defer consumer.Close()

	var got []streaming.Record
	s.Require().NoError(consumer.Consume(s.Ctx, "orders", func(ctx context.Context, r streaming.Record) error {
		got = append(got, r)
		return nil
	}))
	s.Len(got, 2)
	s.Equal("a", string(got[0].Data))
	s.Equal("b", string(got[1].Data))

	// Second consume should not redeliver already-consumed offset.
	got = nil
	s.Require().NoError(consumer.Consume(s.Ctx, "orders", func(ctx context.Context, r streaming.Record) error {
		got = append(got, r)
		return nil
	}))
	s.Empty(got)
}

func (s *ClientSuite) TestPutRecordsBufferFull() {
	client := memory.New(streaming.Config{BufferSize: 2})
	defer client.Close()

	err := client.PutRecords(s.Ctx, []streaming.Record{
		{StreamName: "s", PartitionKey: "k", Data: []byte("1")},
		{StreamName: "s", PartitionKey: "k", Data: []byte("2")},
		{StreamName: "s", PartitionKey: "k", Data: []byte("3")},
	})
	s.Require().Error(err)
	s.True(streaming.IsBufferFull(err))
}

func (s *ClientSuite) TestPutRecord() {
	client, mem, err := s.newClient()
	s.Require().NoError(err)
	defer client.Close()

	data := []byte("hello world")
	s.Require().NoError(client.PutRecord(s.Ctx, "orders", "user-1", data))

	records := mem.GetRecords()
	s.Require().Len(records, 1)
	s.Equal("orders", records[0].StreamName)
	s.Equal("user-1", records[0].PartitionKey)
	s.Equal(data, records[0].Data)

	// Mutating caller's slice must not affect stored record.
	data[0] = 'X'
	s.NotEqual(data, mem.GetRecords()[0].Data)
}

func (s *ClientSuite) TestBufferSizeHonored() {
	client := memory.New(streaming.Config{BufferSize: 2})
	defer client.Close()

	s.Require().NoError(client.PutRecord(s.Ctx, "s", "k", []byte("a")))
	s.Require().NoError(client.PutRecord(s.Ctx, "s", "k", []byte("b")))

	err := client.PutRecord(s.Ctx, "s", "k", []byte("c"))
	s.Require().Error(err)
	s.True(streaming.IsBufferFull(err) || errors.Is(err, streaming.ErrBufferFull))
	s.Len(client.GetRecords(), 2)
}

func (s *ClientSuite) TestUnlimitedBuffer() {
	client := memory.New(streaming.Config{BufferSize: 0})
	defer client.Close()

	for i := 0; i < 50; i++ {
		s.Require().NoError(client.PutRecord(s.Ctx, "s", "k", []byte("x")))
	}
	s.Len(client.GetRecords(), 50)
}

func (s *ClientSuite) TestClose() {
	client, _, err := s.newClient()
	s.Require().NoError(err)
	s.Require().NoError(client.Close())
	s.Require().NoError(client.Close()) // idempotent

	err = client.PutRecord(s.Ctx, "s", "k", []byte("x"))
	s.Require().Error(err)
	s.True(streaming.IsClosed(err) || errors.Is(err, streaming.ErrClosed))
}

func (s *ClientSuite) TestContextCanceled() {
	client, _, err := s.newClient()
	s.Require().NoError(err)
	defer client.Close()

	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	err = client.PutRecord(ctx, "s", "k", []byte("x"))
	s.Require().Error(err)
	s.ErrorIs(err, context.Canceled)
}

func (s *ClientSuite) TestConcurrentPutRecord() {
	client, mem, err := s.newClient()
	s.Require().NoError(err)
	defer client.Close()

	const goroutines = 32
	const perG = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				_ = client.PutRecord(context.Background(), "hot", "pk", []byte("d"))
			}
		}()
	}
	wg.Wait()

	s.Len(mem.GetRecords(), goroutines*perG)
}

func (s *ClientSuite) TestInstrumentedAndResilient() {
	inner := memory.New(streaming.Config{BufferSize: 10})
	client := streaming.NewInstrumentedClient(
		streaming.NewResilientClient(inner, streaming.ResilientConfig{
			CircuitBreakerEnabled: true,
			RetryEnabled:          true,
			RetryMaxAttempts:      2,
		}),
	)
	defer client.Close()

	s.Require().NoError(client.PutRecord(s.Ctx, "s", "k", []byte("ok")))
	s.Len(inner.GetRecords(), 1)
}

func TestMemoryClientSuite(t *testing.T) {
	s := &ClientSuite{
		Suite: test.NewSuite(),
		name:  "memory",
		newClient: func() (streaming.Client, *memory.Client, error) {
			m := memory.New(streaming.Config{BufferSize: 1000})
			return m, m, nil
		},
	}
	suite.Run(t, s)
}

func TestCompileTimeAsserts(t *testing.T) {
	var _ streaming.Client = (*memory.Client)(nil)
	var _ streaming.Consumer = (*memory.Consumer)(nil)
	var _ streaming.Client = (*streaming.InstrumentedClient)(nil)
	var _ streaming.Client = (*streaming.ResilientClient)(nil)
}
