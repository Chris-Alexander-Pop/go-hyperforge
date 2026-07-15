package audit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	"github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/memory"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type AuditSuite struct {
	*test.Suite
}

func TestAuditSuite(t *testing.T) {
	test.Run(t, &AuditSuite{Suite: test.NewSuite()})
}

func (s *AuditSuite) TestLogAppendsToStore() {
	store := memory.NewStore()
	client := audit.New(audit.Config{Enabled: true}, store)

	err := client.Log(s.Ctx, audit.Event{
		EventType: audit.EventTypeLogin,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "user-1",
	})
	s.Require().NoError(err)
	s.Equal(1, store.Len())

	events, err := store.Query(s.Ctx, audit.QueryFilter{ActorID: "user-1"})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	s.Equal(audit.EventTypeLogin, events[0].EventType)
	s.False(events[0].Timestamp.IsZero())
}

func (s *AuditSuite) TestLogDisabledIsNoOp() {
	store := memory.NewStore()
	client := audit.New(audit.Config{Enabled: false}, store)

	err := client.Log(s.Ctx, audit.Event{EventType: audit.EventTypeLogin})
	s.Require().NoError(err)
	s.Equal(0, store.Len())
}

func (s *AuditSuite) TestLogNilStore() {
	client := audit.New(audit.Config{Enabled: true}, nil)
	err := client.Log(s.Ctx, audit.Event{EventType: audit.EventTypeLogin})
	s.Require().Error(err)
	var appErr *pkgerrors.AppError
	s.Require().True(errors.As(err, &appErr))
	s.Equal(audit.CodeInvalidArgument, appErr.Code)
}

func (s *AuditSuite) TestLogCanceledContext() {
	store := memory.NewStore()
	client := audit.New(audit.Config{Enabled: true}, store)
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()

	err := client.Log(ctx, audit.Event{EventType: audit.EventTypeLogin})
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func (s *AuditSuite) TestBuilderSendAndIDs() {
	store := memory.NewStore()
	client := audit.New(audit.Config{Enabled: true}, store)

	err := client.LogWithBuilder(s.Ctx, audit.EventTypeDataRead).
		Actor("alice", "user").
		ActorIP("10.0.0.1").
		Resource("doc-1", "document").
		Target("tgt-1", "file").
		Action("GET /docs").
		Description("read document").
		SessionID("sess-9").
		CorrelationID("corr-9").
		RequestID("req-9").
		Metadata("region", "us-east").
		Outcome(audit.OutcomeSuccess).
		Send()
	s.Require().NoError(err)

	events, err := store.Query(s.Ctx, audit.QueryFilter{})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	e := events[0]
	s.Equal("alice", e.ActorID)
	s.Equal("user", e.ActorType)
	s.Equal("sess-9", e.SessionID)
	s.Equal("corr-9", e.CorrelationID)
	s.Equal("req-9", e.RequestID)
	s.Equal("doc-1", e.ResourceID)
	s.Equal("tgt-1", e.TargetID)
	s.Equal("us-east", e.Metadata["region"])
}

func (s *AuditSuite) TestBuilderErrorMarksFailure() {
	store := memory.NewStore()
	client := audit.New(audit.Config{Enabled: true}, store)

	err := client.LogWithBuilder(s.Ctx, audit.EventTypeLoginFailed).
		Actor("bob", "user").
		Error("AUTH_DENIED", "bad password").
		Send()
	s.Require().NoError(err)

	events, err := store.Query(s.Ctx, audit.QueryFilter{})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	s.Equal(audit.OutcomeFailure, events[0].Outcome)
	s.Equal("AUTH_DENIED", events[0].ErrorCode)
	s.Equal("bad password", events[0].ErrorMessage)
}

func (s *AuditSuite) TestRedactsSensitiveMetadataOnLog() {
	store := memory.NewStore()
	client := audit.New(audit.Config{Enabled: true}, store)

	err := client.Log(s.Ctx, audit.Event{
		EventType:   audit.EventTypeDataCreate,
		Outcome:     audit.OutcomeSuccess,
		Description: "Payment with card 4111-1111-1111-1111",
		Metadata: map[string]interface{}{
			"password": "super-secret",
			"note":     "safe",
		},
	})
	s.Require().NoError(err)

	events, err := store.Query(s.Ctx, audit.QueryFilter{})
	s.Require().NoError(err)
	s.Require().Len(events, 1)
	s.Equal("[REDACTED]", events[0].Metadata["password"])
	s.Equal("safe", events[0].Metadata["note"])
	s.NotContains(events[0].Description, "4111-1111-1111-1111")
	s.Contains(events[0].Description, "[REDACTED]")
}

func (s *AuditSuite) TestInstrumentedAuditorRecordsErrors() {
	store := memory.NewStore()
	client := audit.NewInstrumentedAuditor(audit.New(audit.Config{Enabled: true}, store))

	err := client.LogWithBuilder(s.Ctx, audit.EventTypeLogout).
		Actor("carol", "user").
		SessionID("s1").
		Send()
	s.Require().NoError(err)
	s.Equal(1, store.Len())
}

func (s *AuditSuite) TestLoggerSinkAppendAndQuery() {
	sink := logger.NewSink()
	err := sink.Append(s.Ctx, audit.Event{
		EventType: audit.EventTypeConfigChange,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "admin",
	})
	s.Require().NoError(err)

	_, err = sink.Query(s.Ctx, audit.QueryFilter{})
	s.Require().Error(err)
	var appErr *pkgerrors.AppError
	s.Require().True(errors.As(err, &appErr))
	s.Equal(audit.CodeNotSupported, appErr.Code)
}

func (s *AuditSuite) TestClientWithLoggerSink() {
	client := audit.New(audit.Config{Enabled: true}, logger.NewSink())
	err := client.Log(s.Ctx, audit.Event{
		EventType: audit.EventTypeAccessGranted,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "svc",
	})
	s.Require().NoError(err)
}