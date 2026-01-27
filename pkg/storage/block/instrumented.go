package block

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

// InstrumentedStore wraps a VolumeStore with logging and tracing.
type InstrumentedStore struct {
	next   VolumeStore
	name   string
	tracer trace.Tracer
}

// NewInstrumentedStore creates a new instrumented volume store wrapper.
func NewInstrumentedStore(store VolumeStore, name string) *InstrumentedStore {
	return &InstrumentedStore{
		next:   store,
		name:   name,
		tracer: otel.Tracer("pkg/storage/block"),
	}
}

func (s *InstrumentedStore) startSpan(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := s.tracer.Start(ctx, fmt.Sprintf("%s.%s", s.name, op))
	span.SetAttributes(attrs...)
	return ctx, span
}

func (s *InstrumentedStore) CreateVolume(ctx context.Context, opts CreateVolumeOptions) (*Volume, error) {
	ctx, span := s.startSpan(ctx, "CreateVolume",
		attribute.String("volume.name", opts.Name),
		attribute.Int64("volume.size_gb", opts.SizeGB),
		attribute.String("volume.type", string(opts.VolumeType)),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "creating volume",
		"name", opts.Name,
		"size_gb", opts.SizeGB,
		"type", opts.VolumeType,
	)

	start := time.Now()
	vol, err := s.next.CreateVolume(ctx, opts)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create volume", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("volume.id", vol.ID))
	logger.L().InfoContext(ctx, "created volume", "id", vol.ID, "name", opts.Name, "duration", duration)
	return vol, nil
}

func (s *InstrumentedStore) GetVolume(ctx context.Context, volumeID string) (*Volume, error) {
	ctx, span := s.startSpan(ctx, "GetVolume", attribute.String("volume.id", volumeID))
	defer span.End()

	logger.L().DebugContext(ctx, "getting volume", "id", volumeID)

	vol, err := s.next.GetVolume(ctx, volumeID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get volume", "id", volumeID, "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.String("volume.state", string(vol.State)),
		attribute.Int64("volume.size_gb", vol.SizeGB),
	)
	return vol, nil
}

func (s *InstrumentedStore) ListVolumes(ctx context.Context, opts ListOptions) (*ListResult, error) {
	ctx, span := s.startSpan(ctx, "ListVolumes", attribute.Int("list.limit", opts.Limit))
	defer span.End()

	logger.L().DebugContext(ctx, "listing volumes", "limit", opts.Limit)

	result, err := s.next.ListVolumes(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list volumes", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("list.count", len(result.Volumes)))
	return result, nil
}

func (s *InstrumentedStore) DeleteVolume(ctx context.Context, volumeID string) error {
	ctx, span := s.startSpan(ctx, "DeleteVolume", attribute.String("volume.id", volumeID))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting volume", "id", volumeID)

	err := s.next.DeleteVolume(ctx, volumeID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete volume", "id", volumeID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted volume", "id", volumeID)
	return nil
}

func (s *InstrumentedStore) ResizeVolume(ctx context.Context, volumeID string, opts ResizeVolumeOptions) (*Volume, error) {
	ctx, span := s.startSpan(ctx, "ResizeVolume",
		attribute.String("volume.id", volumeID),
		attribute.Int64("volume.new_size_gb", opts.NewSizeGB),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "resizing volume", "id", volumeID, "new_size_gb", opts.NewSizeGB)

	vol, err := s.next.ResizeVolume(ctx, volumeID, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to resize volume", "id", volumeID, "error", err)
		return nil, err
	}

	logger.L().InfoContext(ctx, "resized volume", "id", volumeID, "new_size_gb", vol.SizeGB)
	return vol, nil
}

func (s *InstrumentedStore) AttachVolume(ctx context.Context, opts AttachVolumeOptions) error {
	ctx, span := s.startSpan(ctx, "AttachVolume",
		attribute.String("volume.id", opts.VolumeID),
		attribute.String("instance.id", opts.InstanceID),
		attribute.String("device", opts.Device),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "attaching volume",
		"volume_id", opts.VolumeID,
		"instance_id", opts.InstanceID,
		"device", opts.Device,
	)

	err := s.next.AttachVolume(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to attach volume",
			"volume_id", opts.VolumeID,
			"instance_id", opts.InstanceID,
			"error", err,
		)
		return err
	}

	logger.L().InfoContext(ctx, "attached volume",
		"volume_id", opts.VolumeID,
		"instance_id", opts.InstanceID,
	)
	return nil
}

func (s *InstrumentedStore) DetachVolume(ctx context.Context, volumeID, instanceID string) error {
	ctx, span := s.startSpan(ctx, "DetachVolume",
		attribute.String("volume.id", volumeID),
		attribute.String("instance.id", instanceID),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "detaching volume", "volume_id", volumeID, "instance_id", instanceID)

	err := s.next.DetachVolume(ctx, volumeID, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to detach volume",
			"volume_id", volumeID,
			"instance_id", instanceID,
			"error", err,
		)
		return err
	}

	logger.L().InfoContext(ctx, "detached volume", "volume_id", volumeID, "instance_id", instanceID)
	return nil
}

func (s *InstrumentedStore) CreateSnapshot(ctx context.Context, opts CreateSnapshotOptions) (*Snapshot, error) {
	ctx, span := s.startSpan(ctx, "CreateSnapshot",
		attribute.String("volume.id", opts.VolumeID),
		attribute.String("snapshot.description", opts.Description),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "creating snapshot", "volume_id", opts.VolumeID)

	start := time.Now()
	snap, err := s.next.CreateSnapshot(ctx, opts)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create snapshot", "volume_id", opts.VolumeID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("snapshot.id", snap.ID))
	logger.L().InfoContext(ctx, "created snapshot", "id", snap.ID, "volume_id", opts.VolumeID, "duration", duration)
	return snap, nil
}

func (s *InstrumentedStore) GetSnapshot(ctx context.Context, snapshotID string) (*Snapshot, error) {
	ctx, span := s.startSpan(ctx, "GetSnapshot", attribute.String("snapshot.id", snapshotID))
	defer span.End()

	logger.L().DebugContext(ctx, "getting snapshot", "id", snapshotID)

	snap, err := s.next.GetSnapshot(ctx, snapshotID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get snapshot", "id", snapshotID, "error", err)
		return nil, err
	}

	return snap, nil
}

func (s *InstrumentedStore) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	ctx, span := s.startSpan(ctx, "DeleteSnapshot", attribute.String("snapshot.id", snapshotID))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting snapshot", "id", snapshotID)

	err := s.next.DeleteSnapshot(ctx, snapshotID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete snapshot", "id", snapshotID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted snapshot", "id", snapshotID)
	return nil
}
