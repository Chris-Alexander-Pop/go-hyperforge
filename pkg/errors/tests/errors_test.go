package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	appErrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"google.golang.org/grpc/codes"
)

type ErrorsSuite struct {
	*test.Suite
}

func TestErrorsSuite(t *testing.T) {
	test.Run(t, &ErrorsSuite{Suite: test.NewSuite()})
}

func (s *ErrorsSuite) TestAppError() {
	originalErr := errors.New("database connection failed")

	e := appErrors.New(appErrors.CodeInternal, "Something went wrong", originalErr)

	s.Equal(appErrors.CodeInternal, e.Code)
	s.Equal("Something went wrong", e.Message)
	s.Equal(originalErr, e.Err)
	s.Equal("[INTERNAL] Something went wrong: database connection failed", e.Error())
	s.Equal(originalErr, errors.Unwrap(e))
}

func (s *ErrorsSuite) TestAppErrorWithoutCause() {
	e := appErrors.New(appErrors.CodeNotFound, "missing", nil)
	s.Equal("[NOT_FOUND] missing", e.Error())
	s.Nil(errors.Unwrap(e))
}

func (s *ErrorsSuite) TestHelpers() {
	err := errors.New("oops")

	notFound := appErrors.NotFound("Not Found", err)
	s.Equal(appErrors.CodeNotFound, notFound.Code)
	s.Equal(http.StatusNotFound, appErrors.HTTPStatus(notFound))

	badReq := appErrors.InvalidArgument("Bad Request", err)
	s.Equal(appErrors.CodeInvalidArgument, badReq.Code)
	s.Equal(http.StatusBadRequest, appErrors.HTTPStatus(badReq))
}

func (s *ErrorsSuite) TestMoreHelpers() {
	err := errors.New("oops")

	unauth := appErrors.Unauthorized("Unauth", err)
	s.Equal(appErrors.CodeUnauthorized, unauth.Code)
	s.Equal(http.StatusUnauthorized, appErrors.HTTPStatus(unauth))

	forbidden := appErrors.Forbidden("Forbidden", err)
	s.Equal(appErrors.CodeForbidden, forbidden.Code)
	s.Equal(http.StatusForbidden, appErrors.HTTPStatus(forbidden))

	conflict := appErrors.Conflict("Conflict", err)
	s.Equal(appErrors.CodeConflict, conflict.Code)
	s.Equal(http.StatusConflict, appErrors.HTTPStatus(conflict))

	internal := appErrors.Internal("Internal", err)
	s.Equal(appErrors.CodeInternal, internal.Code)
	s.Equal(http.StatusInternalServerError, appErrors.HTTPStatus(internal))
}

func (s *ErrorsSuite) TestNewCodes() {
	cause := errors.New("cause")

	deadline := appErrors.DeadlineExceeded("timed out", cause)
	s.Equal(appErrors.CodeDeadlineExceeded, deadline.Code)
	s.Equal(http.StatusGatewayTimeout, appErrors.HTTPStatus(deadline))
	s.Equal(codes.DeadlineExceeded, appErrors.GRPCStatus(deadline).Code())

	unavailable := appErrors.Unavailable("down", cause)
	s.Equal(appErrors.CodeUnavailable, unavailable.Code)
	s.Equal(http.StatusServiceUnavailable, appErrors.HTTPStatus(unavailable))
	s.Equal(codes.Unavailable, appErrors.GRPCStatus(unavailable).Code())

	exhausted := appErrors.ResourceExhausted("quota", cause)
	s.Equal(appErrors.CodeResourceExhausted, exhausted.Code)
	s.Equal(http.StatusTooManyRequests, appErrors.HTTPStatus(exhausted))
	s.Equal(codes.ResourceExhausted, appErrors.GRPCStatus(exhausted).Code())

	canceled := appErrors.Canceled("client gone", cause)
	s.Equal(appErrors.CodeCanceled, canceled.Code)
	s.Equal(appErrors.StatusClientClosedRequest, appErrors.HTTPStatus(canceled))
	s.Equal(codes.Canceled, appErrors.GRPCStatus(canceled).Code())
}

func (s *ErrorsSuite) TestUnimplemented() {
	err := appErrors.Unimplemented("not ready", nil)
	s.Equal(appErrors.CodeUnimplemented, err.Code)
	s.Equal(http.StatusNotImplemented, appErrors.HTTPStatus(err))
	s.Equal(codes.Unimplemented, appErrors.GRPCStatus(err).Code())
}

func (s *ErrorsSuite) TestEmptyMessageDefaults() {
	cases := []struct {
		name string
		err  *appErrors.AppError
		want string
		code string
	}{
		{"NotFound", appErrors.NotFound("", nil), "resource not found", appErrors.CodeNotFound},
		{"InvalidArgument", appErrors.InvalidArgument("", nil), "invalid argument", appErrors.CodeInvalidArgument},
		{"Internal", appErrors.Internal("", nil), "internal server error", appErrors.CodeInternal},
		{"Unauthorized", appErrors.Unauthorized("", nil), "unauthorized", appErrors.CodeUnauthorized},
		{"Forbidden", appErrors.Forbidden("", nil), "forbidden", appErrors.CodeForbidden},
		{"Conflict", appErrors.Conflict("", nil), "conflict", appErrors.CodeConflict},
		{"Unimplemented", appErrors.Unimplemented("", nil), "not implemented", appErrors.CodeUnimplemented},
		{"DeadlineExceeded", appErrors.DeadlineExceeded("", nil), "deadline exceeded", appErrors.CodeDeadlineExceeded},
		{"Unavailable", appErrors.Unavailable("", nil), "service unavailable", appErrors.CodeUnavailable},
		{"ResourceExhausted", appErrors.ResourceExhausted("", nil), "resource exhausted", appErrors.CodeResourceExhausted},
		{"Canceled", appErrors.Canceled("", nil), "canceled", appErrors.CodeCanceled},
	}
	for _, tc := range cases {
		s.Equal(tc.want, tc.err.Message, tc.name)
		s.Equal(tc.code, tc.err.Code, tc.name)
	}
}

func (s *ErrorsSuite) TestWrapPlainError() {
	original := errors.New("root cause")
	wrapped := appErrors.Wrap(original, "context")

	s.Contains(wrapped.Error(), "context: root cause")
	s.Equal(original, errors.Unwrap(wrapped))
	s.False(appErrors.IsCode(wrapped, appErrors.CodeInternal))
	s.Equal("", appErrors.Code(wrapped))
}

func (s *ErrorsSuite) TestWrapPreservesAppError() {
	original := appErrors.NotFound("user missing", nil)
	wrapped := appErrors.Wrap(original, "lookup failed")

	var appErr *appErrors.AppError
	s.True(appErrors.As(wrapped, &appErr))
	s.Equal(appErrors.CodeNotFound, appErr.Code)
	s.Equal("lookup failed", appErr.Message)
	s.True(appErrors.IsCode(wrapped, appErrors.CodeNotFound))
	s.Equal(appErrors.CodeNotFound, appErrors.Code(wrapped))
	s.Equal(http.StatusNotFound, appErrors.HTTPStatus(wrapped))
	s.Equal(codes.NotFound, appErrors.GRPCStatus(wrapped).Code())
}

func (s *ErrorsSuite) TestWrapPreservesNestedAppError() {
	inner := appErrors.Unavailable("db down", nil)
	outer := fmt.Errorf("retry failed: %w", inner)
	wrapped := appErrors.Wrap(outer, "handler failed")

	s.True(appErrors.IsCode(wrapped, appErrors.CodeUnavailable))
	s.Equal(appErrors.CodeUnavailable, appErrors.Code(wrapped))
	s.Equal(http.StatusServiceUnavailable, appErrors.HTTPStatus(wrapped))
	s.Equal(codes.Unavailable, appErrors.GRPCStatus(wrapped).Code())
}

func (s *ErrorsSuite) TestWrapNil() {
	s.Nil(appErrors.Wrap(nil, "ignored"))
}

func (s *ErrorsSuite) TestIsCodeAndCode() {
	err := appErrors.Conflict("exists", nil)
	s.True(appErrors.IsCode(err, appErrors.CodeConflict))
	s.False(appErrors.IsCode(err, appErrors.CodeNotFound))
	s.Equal(appErrors.CodeConflict, appErrors.Code(err))

	plain := errors.New("nope")
	s.False(appErrors.IsCode(plain, appErrors.CodeInternal))
	s.Equal("", appErrors.Code(plain))
	s.False(appErrors.IsCode(nil, appErrors.CodeInternal))
	s.Equal("", appErrors.Code(nil))
}

func (s *ErrorsSuite) TestIsCodeWalksWraps() {
	err := appErrors.Wrap(appErrors.DeadlineExceeded("slow", nil), "op failed")
	s.True(appErrors.IsCode(err, appErrors.CodeDeadlineExceeded))
	s.Equal(appErrors.CodeDeadlineExceeded, appErrors.Code(err))
}

func (s *ErrorsSuite) TestWrappedHTTPAndGRPCMapping() {
	base := appErrors.ResourceExhausted("rate limited", nil)
	wrapped := fmt.Errorf("middleware: %w", base)

	s.Equal(http.StatusTooManyRequests, appErrors.HTTPStatus(wrapped))
	s.Equal(codes.ResourceExhausted, appErrors.GRPCStatus(wrapped).Code())
	s.Equal("rate limited", appErrors.GRPCStatus(wrapped).Message())

	canceledWrapped := fmt.Errorf("ctx: %w", appErrors.Canceled("gone", nil))
	s.Equal(appErrors.StatusClientClosedRequest, appErrors.HTTPStatus(canceledWrapped))
	s.Equal(codes.Canceled, appErrors.GRPCStatus(canceledWrapped).Code())
}

func (s *ErrorsSuite) TestHTTPStatusUnknown() {
	s.Equal(http.StatusInternalServerError, appErrors.HTTPStatus(errors.New("plain")))
	s.Equal(http.StatusInternalServerError, appErrors.HTTPStatus(nil))
}

func (s *ErrorsSuite) TestGRPCStatus() {
	err := appErrors.NotFound("missing", nil)
	st := appErrors.GRPCStatus(err)
	s.Equal("rpc error: code = NotFound desc = missing", st.Err().Error())

	errInvalid := appErrors.InvalidArgument("bad val", nil)
	stInvalid := appErrors.GRPCStatus(errInvalid)
	s.Equal("rpc error: code = InvalidArgument desc = bad val", stInvalid.Err().Error())

	unknown := errors.New("random error")
	stUnknown := appErrors.GRPCStatus(unknown)
	s.Equal("rpc error: code = Unknown desc = random error", stUnknown.Err().Error())
}

func (s *ErrorsSuite) TestGRPCStatusAllCodes() {
	cases := []struct {
		err  *appErrors.AppError
		code codes.Code
	}{
		{appErrors.Unauthorized("u", nil), codes.Unauthenticated},
		{appErrors.Forbidden("f", nil), codes.PermissionDenied},
		{appErrors.Conflict("c", nil), codes.AlreadyExists},
		{appErrors.Internal("i", nil), codes.Internal},
		{appErrors.Unimplemented("n", nil), codes.Unimplemented},
		{appErrors.DeadlineExceeded("d", nil), codes.DeadlineExceeded},
		{appErrors.Unavailable("a", nil), codes.Unavailable},
		{appErrors.ResourceExhausted("r", nil), codes.ResourceExhausted},
		{appErrors.Canceled("x", nil), codes.Canceled},
		{appErrors.Aborted("ab", nil), codes.Aborted},
		{appErrors.FailedPrecondition("fp", nil), codes.FailedPrecondition},
	}
	for _, tc := range cases {
		s.Equal(tc.code, appErrors.GRPCStatus(tc.err).Code(), tc.err.Code)
	}
}

func (s *ErrorsSuite) TestFromHTTPAndFromGRPC() {
	s.True(appErrors.IsCode(appErrors.FromHTTP(http.StatusNotFound, ""), appErrors.CodeNotFound))
	s.True(appErrors.IsCode(appErrors.FromHTTP(http.StatusTooManyRequests, "slow"), appErrors.CodeResourceExhausted))
	s.True(appErrors.IsCode(appErrors.FromHTTP(appErrors.StatusClientClosedRequest, ""), appErrors.CodeCanceled))
	s.True(appErrors.IsCode(appErrors.FromHTTP(http.StatusPreconditionFailed, ""), appErrors.CodeFailedPrecondition))

	s.True(appErrors.IsCode(appErrors.FromGRPC(codes.Aborted, "x"), appErrors.CodeAborted))
	s.True(appErrors.IsCode(appErrors.FromGRPC(codes.FailedPrecondition, "y"), appErrors.CodeFailedPrecondition))
	s.Equal(http.StatusConflict, appErrors.HTTPStatus(appErrors.Aborted("a", nil)))
}

func (s *ErrorsSuite) TestAliases() {
	s.Equal(appErrors.CodeUnauthorized, appErrors.CodeUnauthenticated)
	s.Equal(appErrors.CodeForbidden, appErrors.CodePermissionDenied)
}

func (s *ErrorsSuite) TestIsAndAs() {
	target := errors.New("leaf")
	err := appErrors.Internal("boom", target)

	s.True(appErrors.Is(err, target))
	s.False(appErrors.Is(err, errors.New("other")))

	var appErr *appErrors.AppError
	s.True(appErrors.As(err, &appErr))
	s.Equal(appErrors.CodeInternal, appErr.Code)
}
