package vm

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedVMManager wraps a VMManager with observability.
type InstrumentedVMManager struct {
	next   VMManager
	tracer trace.Tracer
}

// NewInstrumentedVMManager creates a new instrumented VM manager.
func NewInstrumentedVMManager(next VMManager) *InstrumentedVMManager {
	return &InstrumentedVMManager{
		next:   next,
		tracer: otel.Tracer("pkg/compute/vm"),
	}
}

func (i *InstrumentedVMManager) Create(ctx context.Context, opts CreateOptions) (*Instance, error) {
	ctx, span := i.tracer.Start(ctx, "vm.Create", trace.WithAttributes(
		attribute.String("vm.name", opts.Name),
		attribute.String("vm.type", opts.InstanceType),
		attribute.String("vm.image", opts.ImageID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating VM instance", "name", opts.Name, "type", opts.InstanceType)

	instance, err := i.next.Create(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "VM creation failed", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("vm.id", instance.ID))
	logger.L().InfoContext(ctx, "VM created", "id", instance.ID, "name", opts.Name)
	return instance, nil
}

func (i *InstrumentedVMManager) Get(ctx context.Context, instanceID string) (*Instance, error) {
	ctx, span := i.tracer.Start(ctx, "vm.Get", trace.WithAttributes(
		attribute.String("vm.id", instanceID),
	))
	defer span.End()

	instance, err := i.next.Get(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return instance, nil
}

func (i *InstrumentedVMManager) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	ctx, span := i.tracer.Start(ctx, "vm.List", trace.WithAttributes(
		attribute.String("vm.state", string(opts.State)),
		attribute.Int("vm.limit", opts.Limit),
	))
	defer span.End()

	result, err := i.next.List(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("vm.count", len(result.Instances)))
	return result, nil
}

func (i *InstrumentedVMManager) Start(ctx context.Context, instanceID string) error {
	ctx, span := i.tracer.Start(ctx, "vm.Start", trace.WithAttributes(
		attribute.String("vm.id", instanceID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "starting VM", "id", instanceID)

	err := i.next.Start(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "VM start failed", "id", instanceID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedVMManager) Stop(ctx context.Context, instanceID string) error {
	ctx, span := i.tracer.Start(ctx, "vm.Stop", trace.WithAttributes(
		attribute.String("vm.id", instanceID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "stopping VM", "id", instanceID)

	err := i.next.Stop(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "VM stop failed", "id", instanceID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedVMManager) Reboot(ctx context.Context, instanceID string) error {
	ctx, span := i.tracer.Start(ctx, "vm.Reboot", trace.WithAttributes(
		attribute.String("vm.id", instanceID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "rebooting VM", "id", instanceID)

	err := i.next.Reboot(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "VM reboot failed", "id", instanceID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedVMManager) Terminate(ctx context.Context, instanceID string) error {
	ctx, span := i.tracer.Start(ctx, "vm.Terminate", trace.WithAttributes(
		attribute.String("vm.id", instanceID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "terminating VM", "id", instanceID)

	err := i.next.Terminate(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "VM termination failed", "id", instanceID, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedVMManager) UpdateTags(ctx context.Context, instanceID string, tags map[string]string) error {
	ctx, span := i.tracer.Start(ctx, "vm.UpdateTags", trace.WithAttributes(
		attribute.String("vm.id", instanceID),
		attribute.Int("vm.tags_count", len(tags)),
	))
	defer span.End()

	err := i.next.UpdateTags(ctx, instanceID, tags)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (i *InstrumentedVMManager) GetConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	ctx, span := i.tracer.Start(ctx, "vm.GetConsoleOutput", trace.WithAttributes(
		attribute.String("vm.id", instanceID),
	))
	defer span.End()

	output, err := i.next.GetConsoleOutput(ctx, instanceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return output, nil
}

var _ VMManager = (*InstrumentedVMManager)(nil)
