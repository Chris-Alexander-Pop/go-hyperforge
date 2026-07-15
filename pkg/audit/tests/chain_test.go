package audit_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	"github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type ChainSuite struct {
	test.Suite
}

func TestChainSuite(t *testing.T) {
	test.Run(t, new(ChainSuite))
}

func (s *ChainSuite) TestMemoryHashChain() {
	store := memory.NewChainedStore()
	s.Require().NoError(store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogin,
		ActorID:   "a",
		Outcome:   audit.OutcomeSuccess,
	}))
	s.Require().NoError(store.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogout,
		ActorID:   "a",
		Outcome:   audit.OutcomeSuccess,
	}))

	events, err := store.Query(s.Ctx, audit.QueryFilter{})
	s.Require().NoError(err)
	s.Require().Len(events, 2)
	s.Require().NoError(audit.VerifyChain(events))
	s.NotEmpty(events[0].ID)
	s.NotEmpty(events[0].Hash)
	s.Equal("GENESIS", events[0].PrevHash)
	s.Equal(events[0].Hash, events[1].PrevHash)
}

func (s *ChainSuite) TestVerifyChainDetectsTamper() {
	store := memory.NewChainedStore()
	s.Require().NoError(store.Append(s.Ctx, audit.Event{EventType: audit.EventTypeLogin, ActorID: "a"}))
	s.Require().NoError(store.Append(s.Ctx, audit.Event{EventType: audit.EventTypeLogout, ActorID: "a"}))
	events, err := store.Query(s.Ctx, audit.QueryFilter{})
	s.Require().NoError(err)
	events[1].Action = "tampered"
	err = audit.VerifyChain(events)
	s.Require().Error(err)
	s.Contains(err.Error(), audit.CodeChainBroken)
}

func (s *ChainSuite) TestLifecyclePurgeExportErase() {
	store := memory.NewStore()
	old := time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	s.Require().NoError(store.Append(s.Ctx, audit.Event{EventType: audit.EventTypeLogin, ActorID: "alice", Timestamp: old}))
	s.Require().NoError(store.Append(s.Ctx, audit.Event{EventType: audit.EventTypeLogin, ActorID: "bob", Timestamp: now}))
	s.Require().NoError(store.Append(s.Ctx, audit.Event{EventType: audit.EventTypeDataRead, ActorID: "alice", Timestamp: now}))

	n, err := store.Purge(s.Ctx, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Equal(int64(1), n)

	exported, err := store.ExportByActor(s.Ctx, "alice")
	s.Require().NoError(err)
	s.Len(exported, 1)

	erased, err := store.EraseByActor(s.Ctx, "alice")
	s.Require().NoError(err)
	s.Equal(int64(1), erased)
	s.Equal(1, store.Len())
}
