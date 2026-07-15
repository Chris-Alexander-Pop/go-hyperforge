package errors_test

import (
	"fmt"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func Example() {
	// Create a not found error
	err := errors.NotFound("user not found", nil)
	fmt.Println(err.Error())
	// Output: [NOT_FOUND] user not found
}

func ExampleNotFound() {
	err := errors.NotFound("resource does not exist", nil)
	fmt.Println(err.Code)
	// Output: NOT_FOUND
}

func ExampleInvalidArgument() {
	err := errors.InvalidArgument("email is required", nil)
	fmt.Println(err.Code)
	// Output: INVALID_ARGUMENT
}

func ExampleHTTPStatus() {
	err := errors.NotFound("user not found", nil)
	status := errors.HTTPStatus(err)
	fmt.Println(status == http.StatusNotFound)
	// Output: true
}

func ExampleWrap() {
	originalErr := fmt.Errorf("connection refused")
	wrappedErr := errors.Wrap(originalErr, "failed to connect to database")
	fmt.Println(wrappedErr.Error())
	// Output: failed to connect to database: connection refused
}

func ExampleWrap_preserveCode() {
	err := errors.NotFound("user not found", nil)
	wrapped := errors.Wrap(err, "get user")
	fmt.Println(errors.Code(wrapped))
	fmt.Println(errors.IsCode(wrapped, errors.CodeNotFound))
	// Output:
	// NOT_FOUND
	// true
}

func ExampleDeadlineExceeded() {
	err := errors.DeadlineExceeded("request timed out", nil)
	fmt.Println(err.Code)
	fmt.Println(errors.HTTPStatus(err))
	// Output:
	// DEADLINE_EXCEEDED
	// 504
}

func ExampleIsCode() {
	err := errors.Unavailable("upstream down", nil)
	fmt.Println(errors.IsCode(err, errors.CodeUnavailable))
	// Output: true
}

func Example_errorHandling() {
	// Simulate a service function
	getUser := func(id string) error {
		if id == "" {
			return errors.InvalidArgument("user ID is required", nil)
		}
		return errors.NotFound("user not found", nil)
	}

	err := getUser("123")

	// Check error type and convert to HTTP status
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		fmt.Printf("Code: %s, HTTP: %d\n", appErr.Code, errors.HTTPStatus(err))
	}
	// Output: Code: NOT_FOUND, HTTP: 404
}
