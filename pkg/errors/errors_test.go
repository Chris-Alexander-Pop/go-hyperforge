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
	e := appErrors.New(http.StatusInternalServerError, "Something went wrong", originalErr)

	s.Equal(http.StatusInternalServerError, e.Code)
	s.Equal("Something went wrong", e.Message)
	s.Equal(originalErr, e.Err)
	s.Equal("Something went wrong: database connection failed", e.Error())

	// Test Unwrap
	s.Equal(originalErr, errors.Unwrap(e))
}

func (s *ErrorsSuite) TestHelpers() {
	err := errors.New("oops")

	notFound := appErrors.NotFound("Not Found", err)
	s.Equal(http.StatusNotFound, notFound.Code)

	badReq := appErrors.InvalidArgument("Bad Request", err)
	s.Equal(http.StatusBadRequest, badReq.Code)
}
