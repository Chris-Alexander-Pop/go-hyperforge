package messaging_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	auditmem "github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/memory"
	auditmsg "github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/messaging"
	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
	msgmem "github.com/chris-alexander-pop/system-design-library/pkg/messaging/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type FanoutSuite struct {
	test.Suite
	inner    *auditmem.Store
	broker   *msgmem.Broker
	producer messaging.Producer
	fanout   *auditmsg.FanoutStore
}

func (s *FanoutSuite) SetupTest() {
	s.Suite.SetupTest()
	s.inner = auditmem.NewStore()
	s.broker = msgmem.New(msgmem.Config{BufferSize: 16})
	var err error
	s.producer, err = s.broker.Producer("audit.events")
	s.Require().NoError(err)
	s.fanout, err = auditmsg.New(s.inner, s.producer, auditmsg.Config{Topic: "audit.events"})
	s.Require().NoError(err)
}

func (s *FanoutSuite) TearDownTest() {
	if s.broker != nil {
		_ = s.broker.Close()
	}
}

func (s *FanoutSuite) TestAppendFansOut() {
	ctx, cancel := context.WithCancel(s.Ctx)
	defer cancel()

	got := make(chan audit.Event, 1)
	consumer, err := s.broker.Consumer("audit.events", "test-group")
	s.Require().NoError(err)

	go func() {
		_ = consumer.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			var e audit.Event
			if err := json.Unmarshal(msg.Payload, &e); err != nil {
				return err
			}
			got <- e
			return nil
		})
	}()

	s.Require().NoError(s.fanout.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogin,
		ActorID:   "alice",
		Outcome:   audit.OutcomeSuccess,
	}))
	s.Equal(1, s.inner.Len())

	select {
	case e := <-got:
		s.Equal(audit.EventTypeLogin, e.EventType)
		s.Equal("alice", e.ActorID)
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for fanout message")
	}
}

func (s *FanoutSuite) TestQueryDelegates() {
	s.Require().NoError(s.fanout.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeDataRead,
		ActorID:   "bob",
	}))
	events, err := s.fanout.Query(s.Ctx, audit.QueryFilter{ActorID: "bob"})
	s.Require().NoError(err)
	s.Len(events, 1)
}

func (s *FanoutSuite) TestGDPRDelegates() {
	s.Require().NoError(s.fanout.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogin,
		ActorID:   "carol",
	}))
	exported, err := s.fanout.ExportByActor(s.Ctx, "carol")
	s.Require().NoError(err)
	s.Len(exported, 1)
	n, err := s.fanout.EraseByActor(s.Ctx, "carol")
	s.Require().NoError(err)
	s.Equal(int64(1), n)
}

func TestFanoutSuite(t *testing.T) {
	test.Run(t, new(FanoutSuite))
}
