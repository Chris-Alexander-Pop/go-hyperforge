package hypervisor

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedHypervisor wraps a Hypervisor with logging and tracing.
type InstrumentedHypervisor struct {
	next   Hypervisor
	tracer trace.Tracer
}

// NewInstrumentedHypervisor creates a new instrumented hypervisor.
func NewInstrumentedHypervisor(next Hypervisor) *InstrumentedHypervisor {
	return &InstrumentedHypervisor{
		next:   next,
		tracer: otel.Tracer("pkg/cloud/hypervisor"),
	}
}

func (h *InstrumentedHypervisor) CreateVM(ctx context.Context, spec VMSpec) (string, error) {
	ctx, span := h.tracer.Start(ctx, "hypervisor.CreateVM", trace.WithAttributes(
		attribute.String("vm.name", spec.Name),
		attribute.String("vm.instance_type", string(spec.InstanceType)),
		attribute.String("vm.image", spec.Image),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating vm", "name", spec.Name, "type", spec.InstanceType)

	id, err := h.next.CreateVM(ctx, spec)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create vm", "name", spec.Name, "error", err)
		return "", err
	}

	span.SetAttributes(attribute.String("vm.id", id))
	logger.L().InfoContext(ctx, "vm created", "id", id, "name", spec.Name)
	return id, nil
}

func (h *InstrumentedHypervisor) StartVM(ctx context.Context, vmID string) error {
	ctx, span := h.tracer.Start(ctx, "hypervisor.StartVM", trace.WithAttributes(
		attribute.String("vm.id", vmID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "starting vm", "id", vmID)

	err := h.next.StartVM(ctx, vmID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to start vm", "id", vmID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "vm started", "id", vmID)
	return nil
}

func (h *InstrumentedHypervisor) StopVM(ctx context.Context, vmID string) error {
	ctx, span := h.tracer.Start(ctx, "hypervisor.StopVM", trace.WithAttributes(
		attribute.String("vm.id", vmID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "stopping vm", "id", vmID)

	err := h.next.StopVM(ctx, vmID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to stop vm", "id", vmID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "vm stopped", "id", vmID)
	return nil
}

func (h *InstrumentedHypervisor) DeleteVM(ctx context.Context, vmID string) error {
	ctx, span := h.tracer.Start(ctx, "hypervisor.DeleteVM", trace.WithAttributes(
		attribute.String("vm.id", vmID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting vm", "id", vmID)

	err := h.next.DeleteVM(ctx, vmID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete vm", "id", vmID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "vm deleted", "id", vmID)
	return nil
}

func (h *InstrumentedHypervisor) GetVMStatus(ctx context.Context, vmID string) (cloud.InstanceStatus, error) {
	ctx, span := h.tracer.Start(ctx, "hypervisor.GetVMStatus", trace.WithAttributes(
		attribute.String("vm.id", vmID),
	))
	defer span.End()

	status, err := h.next.GetVMStatus(ctx, vmID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return cloud.InstanceStatusUnknown, err
	}

	span.SetAttributes(attribute.String("vm.status", string(status)))
	return status, nil
}

func (h *InstrumentedHypervisor) ListVMs(ctx context.Context) ([]VM, error) { // Fixed reference
	ctx, span := h.tracer.Start(ctx, "hypervisor.ListVMs")
	defer span.End()

	vms, err := h.next.ListVMs(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("vm.count", len(vms)))
	return vms, nil
}
