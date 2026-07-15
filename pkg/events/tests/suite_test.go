package events_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type MemoryBusSuite struct {
	test.Suite
	bus events.Bus
}

func (s *MemoryBusSuite) SetupTest() {
	s.Suite.SetupTest()
	s.bus = memory.New(events.Config{})
}

func (s *MemoryBusSuite) TearDownTest() {
	_ = s.bus.Close()
}

func (s *MemoryBusSuite) TestPublishSubscribe() {
	var received events.Event
	_, err := s.bus.Subscribe(s.Ctx, "users", func(ctx context.Context, e events.Event) error {
		received = e
		return nil
	})
	s.NoError(err)

	err = s.bus.Publish(s.Ctx, "users", events.Event{
		ID:      "123",
		Type:    "user.created",
		Source:  "test",
		Payload: map[string]string{"foo": "bar"},
	})
	s.NoError(err)
	s.Equal("123", received.ID)
	s.False(received.Timestamp.IsZero())
}

func (s *MemoryBusSuite) TestMultiSubscriber() {
	var count atomic.Int32
	for i := 0; i < 3; i++ {
		_, err := s.bus.Subscribe(s.Ctx, "orders", func(ctx context.Context, e events.Event) error {
			count.Add(1)
			return nil
		})
		s.NoError(err)
	}
	s.NoError(s.bus.Publish(s.Ctx, "orders", events.Event{Type: "order.placed", ID: "o1"}))
	s.Equal(int32(3), count.Load())
}

func (s *MemoryBusSuite) TestUnsubscribe() {
	var count atomic.Int32
	sub, err := s.bus.Subscribe(s.Ctx, "payments", func(ctx context.Context, e events.Event) error {
		count.Add(1)
		return nil
	})
	s.NoError(err)
	s.NoError(s.bus.Publish(s.Ctx, "payments", events.Event{Type: "payment.captured"}))
	s.Equal(int32(1), count.Load())
	s.NoError(s.bus.Unsubscribe(s.Ctx, sub))
	s.NoError(s.bus.Publish(s.Ctx, "payments", events.Event{Type: "payment.captured"}))
	s.Equal(int32(1), count.Load())

	err = s.bus.Unsubscribe(s.Ctx, sub)
	s.Error(err)
	var appErr *errors.AppError
	s.True(errors.As(err, &appErr))
	s.Equal(events.CodeSubscriptionNotFound, appErr.Code)
}

func (s *MemoryBusSuite) TestHandlerError() {
	_, err := s.bus.Subscribe(s.Ctx, "users", func(ctx context.Context, e events.Event) error {
		return fmt.Errorf("boom")
	})
	s.NoError(err)
	err = s.bus.Publish(s.Ctx, "users", events.Event{Type: "user.updated"})
	s.Error(err)
	var appErr *errors.AppError
	s.True(errors.As(err, &appErr))
	s.Equal(events.CodeHandlerFailed, appErr.Code)
}

func TestMemoryBusSuite(t *testing.T) {
	test.Run(t, new(MemoryBusSuite))
}
