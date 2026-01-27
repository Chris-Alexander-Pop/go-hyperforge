package loadbalancer

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

// InstrumentedManager wraps a LoadBalancerManager with logging and tracing.
type InstrumentedManager struct {
	next   LoadBalancerManager
	name   string
	tracer trace.Tracer
}

// NewInstrumentedManager creates a new instrumented load balancer manager wrapper.
func NewInstrumentedManager(manager LoadBalancerManager, name string) *InstrumentedManager {
	return &InstrumentedManager{
		next:   manager,
		name:   name,
		tracer: otel.Tracer("pkg/network/loadbalancer"),
	}
}

func (m *InstrumentedManager) startSpan(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := m.tracer.Start(ctx, fmt.Sprintf("%s.%s", m.name, op))
	span.SetAttributes(attrs...)
	return ctx, span
}

func (m *InstrumentedManager) CreateLoadBalancer(ctx context.Context, opts CreateLoadBalancerOptions) (*LoadBalancer, error) {
	ctx, span := m.startSpan(ctx, "CreateLoadBalancer", attribute.String("lb.name", opts.Name))
	defer span.End()

	logger.L().InfoContext(ctx, "creating load balancer", "name", opts.Name, "type", opts.Type)

	start := time.Now()
	lb, err := m.next.CreateLoadBalancer(ctx, opts)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create load balancer", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("lb.id", lb.ID))
	logger.L().InfoContext(ctx, "created load balancer", "id", lb.ID, "name", opts.Name, "duration", duration)
	return lb, nil
}

func (m *InstrumentedManager) GetLoadBalancer(ctx context.Context, id string) (*LoadBalancer, error) {
	ctx, span := m.startSpan(ctx, "GetLoadBalancer", attribute.String("lb.id", id))
	defer span.End()

	logger.L().DebugContext(ctx, "getting load balancer", "id", id)

	lb, err := m.next.GetLoadBalancer(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get load balancer", "id", id, "error", err)
		return nil, err
	}

	return lb, nil
}

func (m *InstrumentedManager) ListLoadBalancers(ctx context.Context) ([]*LoadBalancer, error) {
	ctx, span := m.startSpan(ctx, "ListLoadBalancers")
	defer span.End()

	logger.L().DebugContext(ctx, "listing load balancers")

	lbs, err := m.next.ListLoadBalancers(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list load balancers", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("lb.count", len(lbs)))
	return lbs, nil
}

func (m *InstrumentedManager) DeleteLoadBalancer(ctx context.Context, id string) error {
	ctx, span := m.startSpan(ctx, "DeleteLoadBalancer", attribute.String("lb.id", id))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting load balancer", "id", id)

	err := m.next.DeleteLoadBalancer(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete load balancer", "id", id, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted load balancer", "id", id)
	return nil
}

func (m *InstrumentedManager) CreateListener(ctx context.Context, opts CreateListenerOptions) (*Listener, error) {
	ctx, span := m.startSpan(ctx, "CreateListener",
		attribute.String("lb.id", opts.LoadBalancerID),
		attribute.Int("lb.port", opts.Port),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "creating listener", "lb_id", opts.LoadBalancerID, "port", opts.Port)

	listener, err := m.next.CreateListener(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create listener", "lb_id", opts.LoadBalancerID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("lb.listener_id", listener.ID))
	logger.L().InfoContext(ctx, "created listener", "id", listener.ID)
	return listener, nil
}

func (m *InstrumentedManager) DeleteListener(ctx context.Context, loadBalancerID, listenerID string) error {
	ctx, span := m.startSpan(ctx, "DeleteListener",
		attribute.String("lb.id", loadBalancerID),
		attribute.String("lb.listener_id", listenerID),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "deleting listener", "lb_id", loadBalancerID, "listener_id", listenerID)

	err := m.next.DeleteListener(ctx, loadBalancerID, listenerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete listener", "listener_id", listenerID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted listener", "listener_id", listenerID)
	return nil
}

func (m *InstrumentedManager) CreateTargetPool(ctx context.Context, opts CreateTargetPoolOptions) (*TargetPool, error) {
	ctx, span := m.startSpan(ctx, "CreateTargetPool", attribute.String("lb.pool_name", opts.Name))
	defer span.End()

	logger.L().InfoContext(ctx, "creating target pool", "name", opts.Name)

	pool, err := m.next.CreateTargetPool(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create target pool", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("lb.pool_id", pool.ID))
	logger.L().InfoContext(ctx, "created target pool", "id", pool.ID)
	return pool, nil
}

func (m *InstrumentedManager) GetTargetPool(ctx context.Context, id string) (*TargetPool, error) {
	ctx, span := m.startSpan(ctx, "GetTargetPool", attribute.String("lb.pool_id", id))
	defer span.End()

	logger.L().DebugContext(ctx, "getting target pool", "id", id)

	pool, err := m.next.GetTargetPool(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get target pool", "id", id, "error", err)
		return nil, err
	}

	return pool, nil
}

func (m *InstrumentedManager) DeleteTargetPool(ctx context.Context, id string) error {
	ctx, span := m.startSpan(ctx, "DeleteTargetPool", attribute.String("lb.pool_id", id))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting target pool", "id", id)

	err := m.next.DeleteTargetPool(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete target pool", "id", id, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "deleted target pool", "id", id)
	return nil
}

func (m *InstrumentedManager) AddTarget(ctx context.Context, poolID string, target Target) error {
	ctx, span := m.startSpan(ctx, "AddTarget",
		attribute.String("lb.pool_id", poolID),
		attribute.String("lb.target_address", target.Address),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "adding target", "pool_id", poolID, "address", target.Address)

	err := m.next.AddTarget(ctx, poolID, target)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to add target", "pool_id", poolID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "added target", "pool_id", poolID, "address", target.Address)
	return nil
}

func (m *InstrumentedManager) RemoveTarget(ctx context.Context, poolID, targetID string) error {
	ctx, span := m.startSpan(ctx, "RemoveTarget",
		attribute.String("lb.pool_id", poolID),
		attribute.String("lb.target_id", targetID),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "removing target", "pool_id", poolID, "target_id", targetID)

	err := m.next.RemoveTarget(ctx, poolID, targetID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to remove target", "target_id", targetID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "removed target", "target_id", targetID)
	return nil
}

func (m *InstrumentedManager) GetTargetHealth(ctx context.Context, poolID string) ([]*Target, error) {
	ctx, span := m.startSpan(ctx, "GetTargetHealth", attribute.String("lb.pool_id", poolID))
	defer span.End()

	logger.L().DebugContext(ctx, "getting target health", "pool_id", poolID)

	targets, err := m.next.GetTargetHealth(ctx, poolID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get target health", "pool_id", poolID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("lb.target_count", len(targets)))
	return targets, nil
}

func (m *InstrumentedManager) AddRule(ctx context.Context, listenerID string, rule Rule) (*Rule, error) {
	ctx, span := m.startSpan(ctx, "AddRule", attribute.String("lb.listener_id", listenerID))
	defer span.End()

	logger.L().InfoContext(ctx, "adding rule", "listener_id", listenerID, "priority", rule.Priority)

	result, err := m.next.AddRule(ctx, listenerID, rule)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to add rule", "listener_id", listenerID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("lb.rule_id", result.ID))
	logger.L().InfoContext(ctx, "added rule", "id", result.ID)
	return result, nil
}

func (m *InstrumentedManager) RemoveRule(ctx context.Context, listenerID, ruleID string) error {
	ctx, span := m.startSpan(ctx, "RemoveRule",
		attribute.String("lb.listener_id", listenerID),
		attribute.String("lb.rule_id", ruleID),
	)
	defer span.End()

	logger.L().InfoContext(ctx, "removing rule", "listener_id", listenerID, "rule_id", ruleID)

	err := m.next.RemoveRule(ctx, listenerID, ruleID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to remove rule", "rule_id", ruleID, "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "removed rule", "rule_id", ruleID)
	return nil
}
