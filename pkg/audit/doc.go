/*
Package audit provides structured audit logging for compliance and security.

This package includes:
  - Structured audit events suitable for shipping to log aggregators or SIEM tools
  - Event types for common authentication, authorization, and data operations
  - PII redaction utilities (pattern-based and sensitive field-name matching)
  - Store adapters (in-memory for tests; stdout logger sink for local development)

Durable backends (Kafka, Postgres), retention/GDPR export, and tamper-evident
storage are not provided here; compose a Store adapter for those needs.

Usage:

	import (
		"github.com/chris-alexander-pop/system-design-library/pkg/audit"
		"github.com/chris-alexander-pop/system-design-library/pkg/audit/adapters/memory"
	)

	store := memory.NewStore()
	auditor := audit.NewInstrumentedAuditor(audit.New(audit.Config{Enabled: true}, store))

	_ = auditor.LogWithBuilder(ctx, audit.EventTypeLogin).
		Actor("user-123", "user").
		SessionID("sess-1").
		CorrelationID("corr-1").
		Outcome(audit.OutcomeSuccess).
		Send()
*/
package audit
