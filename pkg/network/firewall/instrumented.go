package firewall

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedFirewallManager wraps a FirewallManager with logging and tracing.
type InstrumentedFirewallManager struct {
	next   FirewallManager
	tracer trace.Tracer
}

// NewInstrumentedFirewallManager creates a new instrumented firewall manager.
func NewInstrumentedFirewallManager(next FirewallManager) *InstrumentedFirewallManager {
	return &InstrumentedFirewallManager{
		next:   next,
		tracer: otel.Tracer("pkg/network/firewall"),
	}
}

func (f *InstrumentedFirewallManager) CreateSecurityGroup(ctx context.Context, spec SecurityGroupSpec) (string, error) {
	ctx, span := f.tracer.Start(ctx, "firewall.CreateSecurityGroup", trace.WithAttributes(
		attribute.String("group.name", spec.Name),
		attribute.String("vpc.id", spec.VPCID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating security group", "name", spec.Name, "vpc_id", spec.VPCID)

	id, err := f.next.CreateSecurityGroup(ctx, spec)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create security group", "error", err)
		return "", err
	}

	span.SetAttributes(attribute.String("group.id", id))
	logger.L().InfoContext(ctx, "security group created", "id", id)
	return id, nil
}

func (f *InstrumentedFirewallManager) DeleteSecurityGroup(ctx context.Context, groupID string) error {
	ctx, span := f.tracer.Start(ctx, "firewall.DeleteSecurityGroup", trace.WithAttributes(
		attribute.String("group.id", groupID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting security group", "id", groupID)

	err := f.next.DeleteSecurityGroup(ctx, groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to delete security group", "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "security group deleted", "id", groupID)
	return nil
}

func (f *InstrumentedFirewallManager) AddRule(ctx context.Context, groupID string, rule Rule) error {
	ctx, span := f.tracer.Start(ctx, "firewall.AddRule", trace.WithAttributes(
		attribute.String("group.id", groupID),
		attribute.String("rule.protocol", rule.Protocol),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "adding rule", "group_id", groupID, "protocol", rule.Protocol)

	err := f.next.AddRule(ctx, groupID, rule)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to add rule", "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "rule added", "group_id", groupID)
	return nil
}

func (f *InstrumentedFirewallManager) RemoveRule(ctx context.Context, groupID string, ruleID string) error {
	ctx, span := f.tracer.Start(ctx, "firewall.RemoveRule", trace.WithAttributes(
		attribute.String("group.id", groupID),
		attribute.String("rule.id", ruleID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "removing rule", "group_id", groupID, "rule_id", ruleID)

	err := f.next.RemoveRule(ctx, groupID, ruleID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to remove rule", "error", err)
		return err
	}

	logger.L().InfoContext(ctx, "rule removed", "group_id", groupID, "rule_id", ruleID)
	return nil
}
