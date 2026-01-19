package errors_test

import (
	"errors"
	"net/http"
	"testing"

	appErrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type ErrorsSuite struct {
	*test.Suite
}

func TestErrorsSuite(t *testing.T) {
	test.Run(t, &ErrorsSuite{Suite: test.NewSuite()})
}

func (s *ErrorsSuite) TestAppError() {
	originalErr := errors.New("database connection failed")

	// Test New Wrapper
	e := appErrors.New(appErrors.CodeInternal, "Something went wrong", originalErr)

	s.Equal(appErrors.CodeInternal, e.Code)
	s.Equal("Something went wrong", e.Message)
	s.Equal(originalErr, e.Err)
	// Update expected error string format: [CODE] Message: Err
	s.Equal("[INTERNAL] Something went wrong: database connection failed", e.Error())

	// Test Unwrap
	s.Equal(originalErr, errors.Unwrap(e))
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

func (s *ErrorsSuite) TestWrap() {
	original := errors.New("root cause")
	wrapped := appErrors.Wrap(original, "context")

	s.Contains(wrapped.Error(), "context: root cause")
	s.Equal(original, errors.Unwrap(wrapped))
}

func (s *ErrorsSuite) TestGRPCStatus() {
	err := appErrors.NotFound("missing", nil)
	st := appErrors.GRPCStatus(err)
	s.Equal("rpc error: code = NotFound desc = missing", st.Err().Error())

	errInvalid := appErrors.InvalidArgument("bad val", nil)
	stInvalid := appErrors.GRPCStatus(errInvalid)
	s.Equal("rpc error: code = InvalidArgument desc = bad val", stInvalid.Err().Error())

	// Test unknown error fallback
	unknown := errors.New("random error")
	stUnknown := appErrors.GRPCStatus(unknown)
	s.Equal("rpc error: code = Unknown desc = random error", stUnknown.Err().Error())
}
