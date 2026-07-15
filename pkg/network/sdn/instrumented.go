package sdn

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedNetworkManager wraps a NetworkManager with logging and tracing.
type InstrumentedNetworkManager struct {
	next   NetworkManager
	tracer trace.Tracer
}

// NewInstrumentedNetworkManager creates a new instrumented network manager.
func NewInstrumentedNetworkManager(next NetworkManager) *InstrumentedNetworkManager {
	return &InstrumentedNetworkManager{
		next:   next,
		tracer: otel.Tracer("pkg/network/sdn"),
	}
}

func (m *InstrumentedNetworkManager) CreateNetwork(ctx context.Context, spec NetworkSpec) (string, error) {
	ctx, span := m.tracer.Start(ctx, "sdn.CreateNetwork", trace.WithAttributes(
		attribute.String("network.name", spec.Name),
		attribute.String("network.cidr", spec.CIDR),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating network", "name", spec.Name, "cidr", spec.CIDR)

	id, err := m.next.CreateNetwork(ctx, spec)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create network", "name", spec.Name, "error", err)
		return "", err
	}

	span.SetAttributes(attribute.String("network.id", id))
	logger.L().InfoContext(ctx, "network created", "id", id)
	return id, nil
}

func (m *InstrumentedNetworkManager) DeleteNetwork(ctx context.Context, networkID string) error {
	ctx, span := m.tracer.Start(ctx, "sdn.DeleteNetwork", trace.WithAttributes(
		attribute.String("network.id", networkID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting network", "id", networkID)

	err := m.next.DeleteNetwork(ctx, networkID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete network", "id", networkID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "network deleted", "id", networkID)
	return nil
}

func (m *InstrumentedNetworkManager) CreateSubnet(ctx context.Context, networkID string, spec SubnetSpec) (string, error) {
	ctx, span := m.tracer.Start(ctx, "sdn.CreateSubnet", trace.WithAttributes(
		attribute.String("network.id", networkID),
		attribute.String("subnet.cidr", spec.CIDR),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating subnet", "network_id", networkID, "cidr", spec.CIDR)

	id, err := m.next.CreateSubnet(ctx, networkID, spec)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create subnet", "network_id", networkID, "error", err)
		return "", err
	}

	span.SetAttributes(attribute.String("subnet.id", id))
	logger.L().InfoContext(ctx, "subnet created", "id", id)
	return id, nil
}

func (m *InstrumentedNetworkManager) DeleteSubnet(ctx context.Context, subnetID string) error {
	ctx, span := m.tracer.Start(ctx, "sdn.DeleteSubnet", trace.WithAttributes(
		attribute.String("subnet.id", subnetID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting subnet", "id", subnetID)

	err := m.next.DeleteSubnet(ctx, subnetID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete subnet", "id", subnetID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "subnet deleted", "id", subnetID)
	return nil
}

func (m *InstrumentedNetworkManager) GetNetwork(ctx context.Context, networkID string) (*Network, error) {
	ctx, span := m.tracer.Start(ctx, "sdn.GetNetwork", trace.WithAttributes(
		attribute.String("network.id", networkID),
	))
	defer span.End()

	network, err := m.next.GetNetwork(ctx, networkID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return network, nil
}
