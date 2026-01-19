package memory_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/streaming"
	"github.com/chris-alexander-pop/system-design-library/pkg/streaming/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type StreamingTestSuite struct {
	test.Suite
	client *memory.Client // Use concrete type to access test helpers
}

func (s *StreamingTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.client = memory.New(streaming.Config{})
}

func (s *StreamingTestSuite) TearDownTest() {
	s.client.Close()
}

func (s *StreamingTestSuite) TestPutRecord() {
	stream := "test-stream"
	key := "partition-1"
	data := []byte("hello world")

	err := s.client.PutRecord(s.Ctx, stream, key, data)
	s.NoError(err)

	// Verify receipt
	records := s.client.GetRecords()
	s.Require().Len(records, 1)
	s.Equal(stream, records[0].StreamName)
	s.Equal(key, records[0].PartitionKey)
	s.Equal(data, records[0].Data)
}

func TestStreamingSuite(t *testing.T) {
	test.Run(t, new(StreamingTestSuite))
}
