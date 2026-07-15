package audit

import (
	"context"
	"time"
)

// Config configures the audit package.
type Config struct {
	// Enabled toggles audit logging on/off.
	Enabled bool `env:"AUDIT_ENABLED" env-default:"true"`

	// RedactConfig for handling sensitive data.
	Redact RedactorConfig
}

// QueryFilter selects audit events by actor, type, and time range.
type QueryFilter struct {
	// ActorID matches Event.ActorID when non-empty.
	ActorID string

	// EventType matches Event.EventType when non-empty.
	EventType EventType

	// Since includes events with Timestamp >= Since when non-zero.
	Since time.Time

	// Until includes events with Timestamp <= Until when non-zero.
	Until time.Time

	// Limit caps the number of results. Zero means no limit.
	Limit int
}

// Store persists and queries audit events.
type Store interface {
	// Append records an audit event.
	Append(ctx context.Context, event Event) error

	// Query returns events matching the filter.
	Query(ctx context.Context, filter QueryFilter) ([]Event, error)
}

// RetentionStore extends Store with retention purge.
type RetentionStore interface {
	Store

	// Purge deletes events with Timestamp strictly before olderThan.
	// Returns the number of removed events.
	Purge(ctx context.Context, olderThan time.Time) (int64, error)
}

// PrivacyStore extends Store with GDPR-oriented export and erase by actor.
type PrivacyStore interface {
	Store

	// ExportByActor returns all events for the given actor ID (subject access).
	ExportByActor(ctx context.Context, actorID string) ([]Event, error)

	// EraseByActor permanently deletes events for the given actor ID (right to erasure).
	// Returns the number of removed events.
	EraseByActor(ctx context.Context, actorID string) (int64, error)
}

// LifecycleStore combines retention and privacy operations.
type LifecycleStore interface {
	RetentionStore
	PrivacyStore
}

// Auditor defines the interface for audit logging.
type Auditor interface {
	// Log records an audit event after redaction.
	Log(ctx context.Context, event Event) error

	// LogWithBuilder starts a fluent event builder.
	LogWithBuilder(ctx context.Context, eventType EventType) *EventBuilder
}

// Client is the default Auditor implementation: redact then Append to a Store.
type Client struct {
	store    Store
	redactor *Redactor
	config   Config
}

// Ensure Client implements Auditor.
var _ Auditor = (*Client)(nil)

// New creates an Auditor that sinks events to the provided store.
// If store is nil, callers should use NewWithLoggerSink or pass an adapter.
func New(cfg Config, store Store) *Client {
	var r *Redactor
	if cfg.Redact.Replacement != "" || len(cfg.Redact.CustomPatterns) > 0 {
		r = NewRedactor(cfg.Redact)
	} else {
		r = NewRedactor(DefaultRedactorConfig())
	}

	return &Client{
		store:    store,
		redactor: r,
		config:   cfg,
	}
}

// Log records an audit event.
func (c *Client) Log(ctx context.Context, event Event) error {
	if !c.config.Enabled {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.store == nil {
		return ErrInvalidArgument("audit store is required", nil)
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if c.redactor != nil {
		event = c.redactor.RedactEvent(event)
	}

	if err := c.store.Append(ctx, event); err != nil {
		return ErrAppendFailed("", err)
	}
	return nil
}

// LogWithBuilder provides a fluent interface for building audit events.
func (c *Client) LogWithBuilder(ctx context.Context, eventType EventType) *EventBuilder {
	return newEventBuilder(c, ctx, eventType)
}

// EventType categorizes audit events.
type EventType string

const (
	// Authentication events
	EventTypeLogin          EventType = "auth.login"
	EventTypeLogout         EventType = "auth.logout"
	EventTypeLoginFailed    EventType = "auth.login_failed"
	EventTypeMFAEnabled     EventType = "auth.mfa_enabled"
	EventTypeMFADisabled    EventType = "auth.mfa_disabled"
	EventTypePasswordChange EventType = "auth.password_change"

	// Authorization events
	EventTypeAccessGranted EventType = "authz.access_granted"
	EventTypeAccessDenied  EventType = "authz.access_denied"

	// Data events
	EventTypeDataCreate EventType = "data.create"
	EventTypeDataRead   EventType = "data.read"
	EventTypeDataUpdate EventType = "data.update"
	EventTypeDataDelete EventType = "data.delete"
	EventTypeDataExport EventType = "data.export"

	// Admin events
	EventTypeConfigChange EventType = "admin.config_change"
	EventTypeUserCreate   EventType = "admin.user_create"
	EventTypeUserModify   EventType = "admin.user_modify"
	EventTypeUserDelete   EventType = "admin.user_delete"
	EventTypeRoleChange   EventType = "admin.role_change"

	// Security events
	EventTypeSecurityAlert      EventType = "security.alert"
	EventTypeRateLimited        EventType = "security.rate_limited"
	EventTypeSuspiciousActivity EventType = "security.suspicious"
)

// Outcome indicates the result of an operation.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeUnknown Outcome = "unknown"
)

// Event represents a structured audit event.
type Event struct {
	// ID uniquely identifies the event (set by hash-chain / durable adapters).
	ID string `json:"id,omitempty"`

	// Required fields
	Timestamp time.Time `json:"timestamp"`
	EventType EventType `json:"event_type"`
	Outcome   Outcome   `json:"outcome"`

	// Actor information
	ActorID        string `json:"actor_id,omitempty"`
	ActorType      string `json:"actor_type,omitempty"` // user, service, system
	ActorIP        string `json:"actor_ip,omitempty"`
	ActorUserAgent string `json:"actor_user_agent,omitempty"`

	// Target information
	TargetID   string `json:"target_id,omitempty"`
	TargetType string `json:"target_type,omitempty"`

	// Resource information
	ResourceID   string `json:"resource_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`

	// Operation details
	Action      string `json:"action,omitempty"`
	Description string `json:"description,omitempty"`

	// Additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Request details
	RequestID     string `json:"request_id,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`

	// Error details (for failures)
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Tamper-evident hash chain (optional; populated by chain-enabled stores).
	// Hash covers the event payload including PrevHash.
	Hash     string `json:"hash,omitempty"`
	PrevHash string `json:"prev_hash,omitempty"`
}

// EventBuilder provides a fluent interface for building audit events.
type EventBuilder struct {
	auditor Auditor
	ctx     context.Context
	event   Event
}

func newEventBuilder(a Auditor, ctx context.Context, eventType EventType) *EventBuilder {
	return &EventBuilder{
		auditor: a,
		ctx:     ctx,
		event: Event{
			Timestamp: time.Now().UTC(),
			EventType: eventType,
			Outcome:   OutcomeSuccess,
		},
	}
}

// Actor sets actor identity fields.
func (b *EventBuilder) Actor(id, actorType string) *EventBuilder {
	b.event.ActorID = id
	b.event.ActorType = actorType
	return b
}

// ActorIP sets the actor IP address.
func (b *EventBuilder) ActorIP(ip string) *EventBuilder {
	b.event.ActorIP = ip
	return b
}

// Target sets target identity fields.
func (b *EventBuilder) Target(id, targetType string) *EventBuilder {
	b.event.TargetID = id
	b.event.TargetType = targetType
	return b
}

// Resource sets resource identity fields.
func (b *EventBuilder) Resource(id, resourceType string) *EventBuilder {
	b.event.ResourceID = id
	b.event.ResourceType = resourceType
	return b
}

// Action sets the action string.
func (b *EventBuilder) Action(action string) *EventBuilder {
	b.event.Action = action
	return b
}

// Description sets a human-readable description.
func (b *EventBuilder) Description(desc string) *EventBuilder {
	b.event.Description = desc
	return b
}

// Outcome sets the operation outcome.
func (b *EventBuilder) Outcome(outcome Outcome) *EventBuilder {
	b.event.Outcome = outcome
	return b
}

// Error marks the event as a failure with an error code and message.
func (b *EventBuilder) Error(code, message string) *EventBuilder {
	b.event.Outcome = OutcomeFailure
	b.event.ErrorCode = code
	b.event.ErrorMessage = message
	return b
}

// Metadata adds a metadata key/value.
func (b *EventBuilder) Metadata(key string, value interface{}) *EventBuilder {
	if b.event.Metadata == nil {
		b.event.Metadata = make(map[string]interface{})
	}
	b.event.Metadata[key] = value
	return b
}

// RequestID sets the request identifier.
func (b *EventBuilder) RequestID(id string) *EventBuilder {
	b.event.RequestID = id
	return b
}

// SessionID sets the session identifier.
func (b *EventBuilder) SessionID(id string) *EventBuilder {
	b.event.SessionID = id
	return b
}

// CorrelationID sets the correlation identifier.
func (b *EventBuilder) CorrelationID(id string) *EventBuilder {
	b.event.CorrelationID = id
	return b
}

// Send persists the built event via the Auditor.
func (b *EventBuilder) Send() error {
	return b.auditor.Log(b.ctx, b.event)
}
