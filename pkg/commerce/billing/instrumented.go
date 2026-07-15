package billing

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedService wraps a Service with logging and tracing.
type InstrumentedService struct {
	next   Service
	tracer trace.Tracer
}

// NewInstrumentedService creates a new InstrumentedService.
func NewInstrumentedService(next Service) *InstrumentedService {
	return &InstrumentedService{
		next:   next,
		tracer: otel.Tracer("pkg/commerce/billing"),
	}
}

func (s *InstrumentedService) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	return s.next.GetPlan(ctx, planID)
}

func (s *InstrumentedService) ListPlans(ctx context.Context) ([]*Plan, error) {
	return s.next.ListPlans(ctx)
}

func (s *InstrumentedService) CreateSubscription(ctx context.Context, customerID string, planID string) (*Subscription, error) {
	ctx, span := s.tracer.Start(ctx, "billing.CreateSubscription", trace.WithAttributes(
		attribute.String("customer.id", customerID),
		attribute.String("plan.id", planID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating subscription", "customer_id", customerID, "plan_id", planID)

	sub, err := s.next.CreateSubscription(ctx, customerID, planID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create subscription", "error", err)
	} else {
		span.SetAttributes(attribute.String("subscription.id", sub.ID))
		logger.L().InfoContext(ctx, "subscription created", "subscription_id", sub.ID)
	}
	return sub, err
}

func (s *InstrumentedService) CancelSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	ctx, span := s.tracer.Start(ctx, "billing.CancelSubscription", trace.WithAttributes(
		attribute.String("subscription.id", subscriptionID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "canceling subscription", "subscription_id", subscriptionID)

	sub, err := s.next.CancelSubscription(ctx, subscriptionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to cancel subscription", "error", err)
	}
	return sub, err
}

func (s *InstrumentedService) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	ctx, span := s.tracer.Start(ctx, "billing.GetSubscription", trace.WithAttributes(
		attribute.String("subscription.id", subscriptionID),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "getting subscription", "subscription_id", subscriptionID)

	sub, err := s.next.GetSubscription(ctx, subscriptionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get subscription", "error", err)
	}
	return sub, err
}

func (s *InstrumentedService) UpgradeSubscription(ctx context.Context, subscriptionID string, newPlanID string) (*Subscription, error) {
	ctx, span := s.tracer.Start(ctx, "billing.UpgradeSubscription", trace.WithAttributes(
		attribute.String("subscription.id", subscriptionID),
		attribute.String("plan.id", newPlanID),
	))
	defer span.End()

	sub, err := s.next.UpgradeSubscription(ctx, subscriptionID, newPlanID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to upgrade subscription", "error", err)
	}
	return sub, err
}

func (s *InstrumentedService) MarkPastDue(ctx context.Context, subscriptionID string) (*Subscription, error) {
	ctx, span := s.tracer.Start(ctx, "billing.MarkPastDue", trace.WithAttributes(
		attribute.String("subscription.id", subscriptionID),
	))
	defer span.End()

	sub, err := s.next.MarkPastDue(ctx, subscriptionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to mark past due", "error", err)
	}
	return sub, err
}

func (s *InstrumentedService) ProcessDunning(ctx context.Context, subscriptionID string) (*DunningResult, error) {
	ctx, span := s.tracer.Start(ctx, "billing.ProcessDunning", trace.WithAttributes(
		attribute.String("subscription.id", subscriptionID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "processing dunning", "subscription_id", subscriptionID)

	result, err := s.next.ProcessDunning(ctx, subscriptionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to process dunning", "error", err)
	} else if result != nil {
		span.SetAttributes(attribute.Int("invoices.transitioned", len(result.Invoices)))
	}
	return result, err
}

func (s *InstrumentedService) CreateInvoice(ctx context.Context, customerID string, amount commerce.Money) (*Invoice, error) {
	ctx, span := s.tracer.Start(ctx, "billing.CreateInvoice", trace.WithAttributes(
		attribute.String("customer.id", customerID),
		attribute.Int64("invoice.amount", amount.Amount),
		attribute.String("invoice.currency", amount.Currency),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating invoice", "customer_id", customerID, "amount", amount.Amount)

	inv, err := s.next.CreateInvoice(ctx, customerID, amount)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to create invoice", "error", err)
	} else {
		span.SetAttributes(attribute.String("invoice.id", inv.ID))
		logger.L().InfoContext(ctx, "invoice created", "invoice_id", inv.ID)
	}
	return inv, err
}

func (s *InstrumentedService) ListInvoices(ctx context.Context, customerID string) ([]*Invoice, error) {
	ctx, span := s.tracer.Start(ctx, "billing.ListInvoices", trace.WithAttributes(
		attribute.String("customer.id", customerID),
	))
	defer span.End()

	invs, err := s.next.ListInvoices(ctx, customerID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return invs, err
}
