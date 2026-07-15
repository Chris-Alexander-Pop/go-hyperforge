package billing

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

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
)
