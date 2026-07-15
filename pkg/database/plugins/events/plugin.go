package events

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Plugin struct {
	bus events.Bus
}

func New(bus events.Bus) *Plugin {
	return &Plugin{bus: bus}
}

func (p *Plugin) Name() string {
	return "events_plugin"
}

func (p *Plugin) Initialize(db *gorm.DB) error {
	// Register callbacks
	if err := db.Callback().Create().After("gorm:create").Register("events:after_create", p.afterCreate); err != nil {
		return fmt.Errorf("failed to register after_create callback: %w", err)
	}
	if err := db.Callback().Update().After("gorm:update").Register("events:after_update", p.afterUpdate); err != nil {
		return fmt.Errorf("failed to register after_update callback: %w", err)
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("events:after_delete", p.afterDelete); err != nil {
		return fmt.Errorf("failed to register after_delete callback: %w", err)
	}
	return nil
}

func (p *Plugin) afterCreate(db *gorm.DB) {
	p.emit(db, "db.created")
}

func (p *Plugin) afterUpdate(db *gorm.DB) {
	p.emit(db, "db.updated")
}

func (p *Plugin) afterDelete(db *gorm.DB) {
	p.emit(db, "db.deleted")
}

func (p *Plugin) emit(db *gorm.DB, eventType string) {
	if db.Error != nil || db.Statement.Schema == nil {
		return
	}

	modelType := db.Statement.Schema.Name // e.g. "User"

	// Try to get ID
	// GORM hooks are powerful but accessing the specific modified fields and values robustly for all cases is complex.
	// For "CDC-Lite", we send the Model data currently in db.Statement.Dest if available.

	payload := db.Statement.Dest

	// Generate a unique ID for the event
	eventID := uuid.New().String()

	evt := events.Event{
		ID:        eventID,
		Type:      eventType,
		Source:    fmt.Sprintf("pkg/database/%s", modelType),
		Timestamp: time.Now(),
		Payload:   payload,
	}

	// Use background context or db context?
	// db.Statement.Context has the request context.
	// We should fire and forget (async) or block?
	// Usually GORM hooks block the transaction. If we fail to publish, should we fail the DB op?
	// For robust CDC, yes (Outbox pattern). For "Notifications", maybe no.
	// We will log errors but NOT fail the DB op to avoid fragility, unless configured.

	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}

	_ = p.bus.Publish(ctx, fmt.Sprintf("db.%s", modelType), evt)
}
