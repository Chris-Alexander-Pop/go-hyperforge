package analytics_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/memory"
	analyticsredis "github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/redis"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type TrackerSuite struct {
	*test.Suite
	newTracker func() (analytics.Tracker, error)
	name       string
}

func (s *TrackerSuite) SetupTest() {
	s.Suite.SetupTest()
}

func (s *TrackerSuite) TestAddCountReset() {
	tracker, err := s.newTracker()
	s.Require().NoError(err)
	defer tracker.Close()

	s.Require().NoError(tracker.Add(s.Ctx, "visitors", "user1"))
	s.Require().NoError(tracker.Add(s.Ctx, "visitors", "user2"))
	s.Require().NoError(tracker.Add(s.Ctx, "visitors", "user1"))

	count, err := tracker.Count(s.Ctx, "visitors")
	s.Require().NoError(err)
	s.Greater(count, uint64(0))

	s.Require().NoError(tracker.Reset(s.Ctx, "visitors"))
	count, err = tracker.Count(s.Ctx, "visitors")
	s.Require().NoError(err)
	s.Equal(uint64(0), count)
}

func (s *TrackerSuite) TestCountMissingReturnsZero() {
	tracker, err := s.newTracker()
	s.Require().NoError(err)
	defer tracker.Close()

	count, err := tracker.Count(s.Ctx, "missing")
	s.Require().NoError(err)
	s.Equal(uint64(0), count)
}

func (s *TrackerSuite) TestMerge() {
	tracker, err := s.newTracker()
	s.Require().NoError(err)
	defer tracker.Close()

	s.Require().NoError(tracker.Add(s.Ctx, "a", "x"))
	s.Require().NoError(tracker.Add(s.Ctx, "a", "y"))
	s.Require().NoError(tracker.Add(s.Ctx, "b", "y"))
	s.Require().NoError(tracker.Add(s.Ctx, "b", "z"))

	s.Require().NoError(tracker.Merge(s.Ctx, "union", "a"))
	s.Require().NoError(tracker.Merge(s.Ctx, "union", "b"))

	count, err := tracker.Count(s.Ctx, "union")
	s.Require().NoError(err)
	s.GreaterOrEqual(count, uint64(2))
}

func (s *TrackerSuite) TestMergeMissingSource() {
	tracker, err := s.newTracker()
	s.Require().NoError(err)
	defer tracker.Close()

	err = tracker.Merge(s.Ctx, "dest", "nope")
	s.Require().Error(err)
	s.True(errors.Is(err, analytics.ErrCounterNotFound) || analytics.IsNotFound(err))
}

func (s *TrackerSuite) TestClose() {
	tracker, err := s.newTracker()
	s.Require().NoError(err)
	s.Require().NoError(tracker.Close())
	s.Require().NoError(tracker.Close()) // idempotent

	err = tracker.Add(s.Ctx, "c", "e")
	s.Require().Error(err)
	s.True(errors.Is(err, analytics.ErrClosed) || errors.IsCode(err, errors.CodeUnavailable))
}

func (s *TrackerSuite) TestConcurrentAddCount() {
	tracker, err := s.newTracker()
	s.Require().NoError(err)
	defer tracker.Close()

	const goroutines = 32
	const perG = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				elem := fmt.Sprintf("u-%d-%d", id, i)
				_ = tracker.Add(context.Background(), "hot", elem)
				_, _ = tracker.Count(context.Background(), "hot")
			}
		}(g)
	}
	wg.Wait()

	count, err := tracker.Count(s.Ctx, "hot")
	s.Require().NoError(err)
	s.Greater(count, uint64(0))
}

func TestMemoryTrackerSuite(t *testing.T) {
	s := &TrackerSuite{
		Suite: test.NewSuite(),
		name:  "memory",
		newTracker: func() (analytics.Tracker, error) {
			return memory.New(analytics.DefaultConfig())
		},
	}
	suite.Run(t, s)
}

func TestRedisTrackerSuite(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	s := &TrackerSuite{
		Suite: test.NewSuite(),
		name:  "redis",
		newTracker: func() (analytics.Tracker, error) {
			client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
			return analyticsredis.New(client, analyticsredis.WithKeyPrefix("test:hll:")), nil
		},
	}
	suite.Run(t, s)
}

func TestConfigNormalize(t *testing.T) {
	cfg, err := analytics.Config{}.Normalize()
	if err != nil {
		t.Fatalf("zero config should default: %v", err)
	}
	if cfg.Precision != 14 {
		t.Fatalf("expected default precision 14, got %d", cfg.Precision)
	}

	_, err = analytics.Config{Precision: 3}.Normalize()
	if err == nil {
		t.Fatal("expected validation error for precision 3")
	}
	if !errors.IsCode(err, errors.CodeInvalidArgument) {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}

	_, err = analytics.Config{Precision: 17}.Normalize()
	if err == nil {
		t.Fatal("expected validation error for precision 17")
	}

	cfg, err = analytics.Config{Precision: 10}.Normalize()
	if err != nil {
		t.Fatalf("precision 10 should be valid: %v", err)
	}
	if cfg.Precision != 10 {
		t.Fatalf("expected 10, got %d", cfg.Precision)
	}
}

func TestMemoryInvalidConfig(t *testing.T) {
	_, err := memory.New(analytics.Config{Precision: 2})
	if err == nil {
		t.Fatal("expected error for invalid precision")
	}
}

func TestInstrumentedTracker(t *testing.T) {
	inner, err := memory.New(analytics.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer inner.Close()

	tracker := analytics.NewInstrumentedTracker(inner)
	ctx := context.Background()
	if err := tracker.Add(ctx, "i", "a"); err != nil {
		t.Fatal(err)
	}
	n, err := tracker.Count(ctx, "i")
	if err != nil || n == 0 {
		t.Fatalf("count=%d err=%v", n, err)
	}
	if err := tracker.Merge(ctx, "j", "i"); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Reset(ctx, "i"); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Close(); err != nil {
		t.Fatal(err)
	}
}
