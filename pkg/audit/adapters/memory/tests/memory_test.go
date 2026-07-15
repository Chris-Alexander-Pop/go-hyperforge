package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	"github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/memory"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type MemoryStoreSuite struct {
	*test.Suite
	store *memory.Store
}

func TestMemoryStoreSuite(t *testing.T) {
	test.Run(t, &MemoryStoreSuite{Suite: test.NewSuite()})
}

func (s *MemoryStoreSuite) SetupTest() {
	s.Suite.SetupTest()
	s.store = memory.NewStore()
}

func (s *MemoryStoreSuite) appendEvent(e audit.Event) {
	s.T().Helper()
	s.Require().NoError(s.store.Append(s.Ctx, e))
}

func (s *MemoryStoreSuite) TestAppendRequiresEventType() {
	err := s.store.Append(s.Ctx, audit.Event{})
	s.Require().Error(err)
	var appErr *pkgerrors.AppError
	s.Require().True(errors.As(err, &appErr))
	s.Equal(audit.CodeInvalidArgument, appErr.Code)
}

func (s *MemoryStoreSuite) TestAppendSetsTimestampAndCopiesMetadata() {
	meta := map[string]interface{}{"k": "v"}
	s.appendEvent(audit.Event{
		EventType: audit.EventTypeLogin,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "a1",
		Metadata:  meta,
	})
	meta["k"] = "mutated"

	events, err := s.store.Query(s.Ctx, audit.QueryFilter{})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	s.False(events[0].Timestamp.IsZero())
	s.Equal("v", events[0].Metadata["k"])
}

func (s *MemoryStoreSuite) TestQueryFilterByActor() {
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogin, ActorID: "alice"})
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogout, ActorID: "bob"})
	s.appendEvent(audit.Event{EventType: audit.EventTypeDataRead, ActorID: "alice"})

	events, err := s.store.Query(s.Ctx, audit.QueryFilter{ActorID: "alice"})
	s.Require().NoError(err)
	s.Require().Len(events, 2)
	for _, e := range events {
		s.Equal("alice", e.ActorID)
	}
}

func (s *MemoryStoreSuite) TestQueryFilterByEventType() {
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogin, ActorID: "a"})
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogout, ActorID: "a"})
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogin, ActorID: "b"})

	events, err := s.store.Query(s.Ctx, audit.QueryFilter{EventType: audit.EventTypeLogin})
	s.Require().NoError(err)
	s.Require().Len(events, 2)
	for _, e := range events {
		s.Equal(audit.EventTypeLogin, e.EventType)
	}
}

func (s *MemoryStoreSuite) TestQueryFilterByTimeRange() {
	t0 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	t2 := t0.Add(2 * time.Hour)

	s.appendEvent(audit.Event{EventType: audit.EventTypeLogin, Timestamp: t0, ActorID: "a"})
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogin, Timestamp: t1, ActorID: "a"})
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogin, Timestamp: t2, ActorID: "a"})

	events, err := s.store.Query(s.Ctx, audit.QueryFilter{
		Since: t1,
		Until: t1,
	})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	s.True(events[0].Timestamp.Equal(t1))

	events, err = s.store.Query(s.Ctx, audit.QueryFilter{Since: t1})
	s.Require().NoError(err)
	s.Len(events, 2)

	events, err = s.store.Query(s.Ctx, audit.QueryFilter{Until: t0})
	s.Require().NoError(err)
	s.Len(events, 1)
}

func (s *MemoryStoreSuite) TestQueryLimit() {
	for i := 0; i < 5; i++ {
		s.appendEvent(audit.Event{EventType: audit.EventTypeDataRead, ActorID: "a"})
	}
	events, err := s.store.Query(s.Ctx, audit.QueryFilter{Limit: 2})
	s.Require().NoError(err)
	s.Len(events, 2)
}

func (s *MemoryStoreSuite) TestQueryNegativeLimit() {
	_, err := s.store.Query(s.Ctx, audit.QueryFilter{Limit: -1})
	s.Require().Error(err)
	var appErr *pkgerrors.AppError
	s.Require().True(errors.As(err, &appErr))
	s.Equal(audit.CodeInvalidArgument, appErr.Code)
}

func (s *MemoryStoreSuite) TestQueryCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	_, err := s.store.Query(ctx, audit.QueryFilter{})
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func (s *MemoryStoreSuite) TestAppendCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	err := s.store.Append(ctx, audit.Event{EventType: audit.EventTypeLogin})
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func (s *MemoryStoreSuite) TestCombinedFilters() {
	now := time.Now().UTC()
	s.appendEvent(audit.Event{
		EventType: audit.EventTypeDataExport,
		ActorID:   "alice",
		Timestamp: now,
	})
	s.appendEvent(audit.Event{
		EventType: audit.EventTypeDataExport,
		ActorID:   "bob",
		Timestamp: now,
	})
	s.appendEvent(audit.Event{
		EventType: audit.EventTypeDataRead,
		ActorID:   "alice",
		Timestamp: now,
	})

	events, err := s.store.Query(s.Ctx, audit.QueryFilter{
		ActorID:   "alice",
		EventType: audit.EventTypeDataExport,
		Limit:     10,
	})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	s.Equal("alice", events[0].ActorID)
	s.Equal(audit.EventTypeDataExport, events[0].EventType)
}

func (s *MemoryStoreSuite) TestInstrumentedStore() {
	base := memory.NewStore()
	store := audit.NewInstrumentedStore(base)

	err := store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeUserCreate,
		ActorID:   "admin",
	})
	s.Require().NoError(err)

	events, err := store.Query(s.Ctx, audit.QueryFilter{ActorID: "admin"})
	s.Require().NoError(err)
	s.Len(events, 1)
}

func (s *MemoryStoreSuite) TestLen() {
	s.Equal(0, s.store.Len())
	s.appendEvent(audit.Event{EventType: audit.EventTypeLogin})
	s.Equal(1, s.store.Len())
}
