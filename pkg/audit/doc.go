/*
Package audit provides structured audit logging for compliance and security.

This package includes:
  - Structured audit events suitable for shipping to log aggregators or SIEM tools
  - Event types for common authentication, authorization, and data operations
  - PII redaction utilities (pattern-based and sensitive field-name matching)
  - Store adapters: memory, stdout logger, durable SQL/Postgres, messaging fanout
  - Optional tamper-evident hash chaining (Hash / PrevHash)
  - Retention purge and GDPR Export/Erase-by-actor on LifecycleStore adapters

Usage:

	import (
		"github.com/chris-alexander-pop/system-design-library/pkg/audit"
		"github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/memory"
	)

	store := memory.NewChainedStore()
	auditor := audit.NewInstrumentedAuditor(audit.New(audit.Config{Enabled: true}, store))

	_ = auditor.LogWithBuilder(ctx, audit.EventTypeLogin).
		Actor("user-123", "user").
		SessionID("sess-1").
		CorrelationID("corr-1").
		Outcome(audit.OutcomeSuccess).
		Send()
*/
package audit
