package dns

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedManager wraps a DNSManager with logging and tracing.
type InstrumentedManager struct {
	next   DNSManager
	name   string
	tracer trace.Tracer
}

// NewInstrumentedManager creates a new instrumented DNS manager wrapper.
func NewInstrumentedManager(manager DNSManager, name string) *InstrumentedManager {
	return &InstrumentedManager{
		next:   manager,
		name:   name,
		tracer: otel.Tracer("pkg/network/dns"),
	}
}

func (m *InstrumentedManager) startSpan(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, fmt.Sprintf("%s.%s", m.name, op))
	span.SetAttributes(attrs...)
	return ctx, span
}

func (m *InstrumentedManager) CreateZone(ctx context.Context, opts CreateZoneOptions) (*Zone, error) {
	ctx, span := m.startSpan(ctx, "CreateZone", attribute.String("dns.zone", opts.Name))
	defer span.End()

	logger.L().InfoContext(ctx, "creating DNS zone", "name", opts.Name)

	zone, err := m.next.CreateZone(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create zone", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("dns.zone_id", zone.ID))
	logger.L().InfoContext(ctx, "created DNS zone", "id", zone.ID, "name", opts.Name)
	return zone, nil
}

func (m *InstrumentedManager) GetZone(ctx context.Context, zoneID string) (*Zone, error) {
	ctx, span := m.startSpan(ctx, "GetZone", attribute.String("dns.zone_id", zoneID))
	defer span.End()

	logger.L().DebugContext(ctx, "getting DNS zone", "zone_id", zoneID)

	zone, err := m.next.GetZone(ctx, zoneID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get zone", "zone_id", zoneID, "error", err)
		return nil, err
	}

	return zone, nil
}

func (m *InstrumentedManager) ListZones(ctx context.Context) ([]*Zone, error) {
	ctx, span := m.startSpan(ctx, "ListZones")
	defer span.End()

	logger.L().DebugContext(ctx, "listing DNS zones")

	zones, err := m.next.ListZones(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list zones", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("dns.zone_count", len(zones)))
	return zones, nil
}

func (m *InstrumentedManager) DeleteZone(ctx context.Context, zoneID string) error {
	ctx, span := m.startSpan(ctx, "DeleteZone", attribute.String("dns.zone_id", zoneID))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting DNS zone", "zone_id", zoneID)

	err := m.next.DeleteZone(ctx, zoneID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete zone", "zone_id", zoneID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted DNS zone", "zone_id", zoneID)
	return nil
}

func (m *InstrumentedManager) CreateRecord(ctx context.Context, opts CreateRecordOptions) (*Record, error) {
	ctx, span := m.startSpan(ctx, "CreateRecord",
		attribute.String("dns.zone_id", opts.ZoneID),
		attribute.String("dns.record_name", opts.Name),
		attribute.String("dns.record_type", string(opts.Type)),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "creating DNS record",
		"zone_id", opts.ZoneID,
		"name", opts.Name,
		"type", opts.Type,
	)

	start := time.Now()
	record, err := m.next.CreateRecord(ctx, opts)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create record", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("dns.record_id", record.ID))
	logger.L().InfoContext(ctx, "created DNS record",
		"id", record.ID,
		"name", opts.Name,
		"duration", duration,
	)
	return record, nil
}

func (m *InstrumentedManager) GetRecord(ctx context.Context, zoneID, recordID string) (*Record, error) {
	ctx, span := m.startSpan(ctx, "GetRecord",
		attribute.String("dns.zone_id", zoneID),
		attribute.String("dns.record_id", recordID),
	)
	defer span.End()

	logger.L().DebugContext(ctx, "getting DNS record", "zone_id", zoneID, "record_id", recordID)

	record, err := m.next.GetRecord(ctx, zoneID, recordID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get record", "record_id", recordID, "error", err)
		return nil, err
	}

	return record, nil
}

func (m *InstrumentedManager) ListRecords(ctx context.Context, zoneID string, opts ListRecordsOptions) (*ListRecordsResult, error) {
	ctx, span := m.startSpan(ctx, "ListRecords", attribute.String("dns.zone_id", zoneID))
	defer span.End()

	logger.L().DebugContext(ctx, "listing DNS records", "zone_id", zoneID)

	result, err := m.next.ListRecords(ctx, zoneID, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list records", "zone_id", zoneID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("dns.record_count", len(result.Records)))
	return result, nil
}

func (m *InstrumentedManager) UpdateRecord(ctx context.Context, zoneID, recordID string, opts UpdateRecordOptions) (*Record, error) {
	ctx, span := m.startSpan(ctx, "UpdateRecord",
		attribute.String("dns.zone_id", zoneID),
		attribute.String("dns.record_id", recordID),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "updating DNS record", "zone_id", zoneID, "record_id", recordID)

	record, err := m.next.UpdateRecord(ctx, zoneID, recordID, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to update record", "record_id", recordID, "error", err)
		return nil, err
	}

	logger.L().InfoContext(ctx, "updated DNS record", "record_id", recordID)
	return record, nil
}

func (m *InstrumentedManager) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	ctx, span := m.startSpan(ctx, "DeleteRecord",
		attribute.String("dns.zone_id", zoneID),
		attribute.String("dns.record_id", recordID),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "deleting DNS record", "zone_id", zoneID, "record_id", recordID)

	err := m.next.DeleteRecord(ctx, zoneID, recordID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete record", "record_id", recordID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted DNS record", "record_id", recordID)
	return nil
}

func (m *InstrumentedManager) LookupRecord(ctx context.Context, name string, recordType RecordType) ([]*Record, error) {
	ctx, span := m.startSpan(ctx, "LookupRecord",
		attribute.String("dns.name", name),
		attribute.String("dns.type", string(recordType)),
	)
	defer span.End()

	logger.L().DebugContext(ctx, "looking up DNS record", "name", name, "type", recordType)

	records, err := m.next.LookupRecord(ctx, name, recordType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to lookup record", "name", name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("dns.result_count", len(records)))
	return records, nil
}
