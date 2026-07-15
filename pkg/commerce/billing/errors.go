package billing

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrPlanNotFound indicates the plan ID is unknown.
	ErrPlanNotFound = errors.NotFound("plan not found", nil)

	// ErrSubscriptionNotFound indicates the subscription does not exist.
	ErrSubscriptionNotFound = errors.NotFound("subscription not found", nil)

	// ErrSubscriptionCanceled indicates the subscription is already canceled.
	ErrSubscriptionCanceled = errors.Conflict("subscription already canceled", nil)

	// ErrInvalidPlan indicates plan data is invalid.
	ErrInvalidPlan = errors.InvalidArgument("invalid plan", nil)

	// ErrSamePlan indicates an upgrade target matches the current plan.
	ErrSamePlan = errors.InvalidArgument("subscription already on this plan", nil)

	// ErrCurrencyMismatch indicates plan amounts use different currencies.
	ErrCurrencyMismatch = errors.InvalidArgument("plan currency mismatch", nil)

	// ErrInvalidPeriod indicates a billing period end is not after its start.
	ErrInvalidPeriod = errors.InvalidArgument("invalid billing period", nil)

	// ErrInvoiceNotFound indicates the invoice does not exist.
	ErrInvoiceNotFound = errors.NotFound("invoice not found", nil)
)
