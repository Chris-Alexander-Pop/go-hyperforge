package controller

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedVolumeController wraps a VolumeController with logging and tracing.
type InstrumentedVolumeController struct {
	next   VolumeController
	tracer trace.Tracer
}

// NewInstrumentedVolumeController creates a new instrumented volume controller.
func NewInstrumentedVolumeController(next VolumeController) *InstrumentedVolumeController {
	return &InstrumentedVolumeController{
		next:   next,
		tracer: otel.Tracer("pkg/storage/controller"),
	}
}

func (c *InstrumentedVolumeController) CreateVolume(ctx context.Context, spec VolumeSpec) (string, error) {
	ctx, span := c.tracer.Start(ctx, "controller.CreateVolume", trace.WithAttributes(
		attribute.String("volume.name", spec.Name),
		attribute.Int("volume.size_gb", spec.SizeGB),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating volume", "name", spec.Name, "size_gb", spec.SizeGB)

	id, err := c.next.CreateVolume(ctx, spec)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create volume", "name", spec.Name, "error", err)
		return "", err
	}

	span.SetAttributes(attribute.String("volume.id", id))
	logger.L().InfoContext(ctx, "volume created", "id", id)
	return id, nil
}

func (c *InstrumentedVolumeController) DeleteVolume(ctx context.Context, volumeID string) error {
	ctx, span := c.tracer.Start(ctx, "controller.DeleteVolume", trace.WithAttributes(
		attribute.String("volume.id", volumeID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting volume", "id", volumeID)

	err := c.next.DeleteVolume(ctx, volumeID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete volume", "id", volumeID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "volume deleted", "id", volumeID)
	return nil
}

func (c *InstrumentedVolumeController) AttachVolume(ctx context.Context, volumeID string, nodeID string) error {
	ctx, span := c.tracer.Start(ctx, "controller.AttachVolume", trace.WithAttributes(
		attribute.String("volume.id", volumeID),
		attribute.String("node.id", nodeID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "attaching volume", "id", volumeID, "node_id", nodeID)

	err := c.next.AttachVolume(ctx, volumeID, nodeID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to attach volume", "id", volumeID, "node_id", nodeID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "volume attached", "id", volumeID, "node_id", nodeID)
	return nil
}

func (c *InstrumentedVolumeController) DetachVolume(ctx context.Context, volumeID string) error {
	ctx, span := c.tracer.Start(ctx, "controller.DetachVolume", trace.WithAttributes(
		attribute.String("volume.id", volumeID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "detaching volume", "id", volumeID)

	err := c.next.DetachVolume(ctx, volumeID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to detach volume", "id", volumeID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "volume detached", "id", volumeID)
	return nil
}

func (c *InstrumentedVolumeController) ResizeVolume(ctx context.Context, volumeID string, newSizeGB int) error {
	ctx, span := c.tracer.Start(ctx, "controller.ResizeVolume", trace.WithAttributes(
		attribute.String("volume.id", volumeID),
		attribute.Int("volume.new_size_gb", newSizeGB),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "resizing volume", "id", volumeID, "new_size_gb", newSizeGB)

	err := c.next.ResizeVolume(ctx, volumeID, newSizeGB)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to resize volume", "id", volumeID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "volume resized", "id", volumeID)
	return nil
}

func (c *InstrumentedVolumeController) GetVolume(ctx context.Context, volumeID string) (*Volume, error) {
	ctx, span := c.tracer.Start(ctx, "controller.GetVolume", trace.WithAttributes(
		attribute.String("volume.id", volumeID),
	))
	defer span.End()

	vol, err := c.next.GetVolume(ctx, volumeID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return vol, nil
}
