package mapreduce

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestMapReduce verifies the correctness of the MapReduce implementation.
func TestMapReduce(t *testing.T) {
	inputData := map[string]interface{}{
		"doc1": "foo bar foo",
		"doc2": "bar baz",
	}

	mapper := func(key string, value interface{}, out chan<- KeyValue) {
		text := value.(string)
		words := strings.Fields(text)
		for _, w := range words {
			out <- KeyValue{Key: w, Value: 1}
		}
	}

	reducer := func(key string, values []interface{}, out chan<- interface{}) {
		count := 0
		for _, v := range values {
			count += v.(int)
		}
		out <- count
	}

	job := NewJob(mapper, reducer, inputData, 2)
	results, err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("Job failed: %v", err)
	}

	expected := map[string]int{
		"foo": 2,
		"bar": 2,
		"baz": 1,
	}

	for k, v := range expected {
		res, ok := results[k]
		if !ok {
			t.Errorf("Expected key %s not found", k)
			continue
		}
		if len(res) != 1 {
			t.Errorf("Expected 1 result for key %s, got %d", k, len(res))
			continue
		}
		if res[0].(int) != v {
			t.Errorf("Expected value %d for key %s, got %v", v, k, res[0])
		}
	}
}

// BenchmarkMapReduce measures the performance of the MapReduce implementation.
func BenchmarkMapReduce(b *testing.B) {
	// Setup input data
	inputData := make(map[string]interface{})
	numInputs := 100
	for i := 0; i < numInputs; i++ {
		inputData[fmt.Sprintf("doc-%d", i)] = "data"
	}

	// Mapper generates multiple unique keys per input to create a large number of keys
	mapper := func(key string, value interface{}, out chan<- KeyValue) {
		var docId int
		if _, err := fmt.Sscanf(key, "doc-%d", &docId); err != nil {
			return
		}
		// Emit 100 keys per input.
		// Construct keys such that we have many unique keys.
		for i := 0; i < 100; i++ {
			k := fmt.Sprintf("key-%d-%d", docId, i)
			out <- KeyValue{Key: k, Value: 1}
		}
	}

	// Reducer is simple
	reducer := func(key string, values []interface{}, out chan<- interface{}) {
		// Simulate slight processing time
		time.Sleep(1 * time.Microsecond)
		count := 0
		for _, v := range values {
			count += v.(int)
		}
		out <- count
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use 10 workers
		job := NewJob(mapper, reducer, inputData, 10)
		_, err := job.Run(context.Background())
		if err != nil {
			b.Fatalf("Job failed: %v", err)
		}
	}
}
