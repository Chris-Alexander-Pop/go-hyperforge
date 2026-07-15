package sql_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	auditsql "github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/sql"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	_ "modernc.org/sqlite"
)

type SQLStoreSuite struct {
	test.Suite
	db    *sql.DB
	store *auditsql.Store
}

func (s *SQLStoreSuite) SetupTest() {
	s.Suite.SetupTest()
	db, err := sql.Open("sqlite", "file:audit_test?mode=memory&cache=shared")
	s.Require().NoError(err)
	s.db = db
	store, err := auditsql.New(db, auditsql.Config{Dialect: auditsql.DialectSQLite, HashChain: true})
	s.Require().NoError(err)
	s.Require().NoError(store.Migrate(s.Ctx))
	s.store = store
}

func (s *SQLStoreSuite) TearDownTest() {
	if s.db != nil {
		_ = s.db.Close()
	}
}

func (s *SQLStoreSuite) TestAppendQueryChain() {
	s.Require().NoError(s.store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogin,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "alice",
	}))
	s.Require().NoError(s.store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogout,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "alice",
	}))

	events, err := s.store.Query(s.Ctx, audit.QueryFilter{ActorID: "alice"})
	s.Require().NoError(err)
	s.Require().Len(events, 2)
	s.Require().NoError(audit.VerifyChain(events))
	s.Equal("GENESIS", events[0].PrevHash)
	s.Equal(events[0].Hash, events[1].PrevHash)
}

func (s *SQLStoreSuite) TestPurgeAndGDPR() {
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	s.Require().NoError(s.store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogin,
		ActorID:   "alice",
		Timestamp: old,
	}))
	s.Require().NoError(s.store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogin,
		ActorID:   "bob",
		Timestamp: now,
	}))
	s.Require().NoError(s.store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeDataRead,
		ActorID:   "alice",
		Timestamp: now,
	}))

	n, err := s.store.Purge(s.Ctx, time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Equal(int64(1), n)

	exported, err := s.store.ExportByActor(s.Ctx, "alice")
	s.Require().NoError(err)
	s.Len(exported, 1)

	erased, err := s.store.EraseByActor(s.Ctx, "alice")
	s.Require().NoError(err)
	s.Equal(int64(1), erased)

	events, err := s.store.Query(s.Ctx, audit.QueryFilter{})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	s.Equal("bob", events[0].ActorID)
}

func (s *SQLStoreSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	err := s.store.Append(ctx, audit.Event{EventType: audit.EventTypeLogin})
	s.Require().Error(err)
}

func TestSQLStoreSuite(t *testing.T) {
	test.Run(t, new(SQLStoreSuite))
}
