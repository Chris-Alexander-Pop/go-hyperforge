package validator

import (
	"fmt"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	playground "github.com/go-playground/validator/v10"
)

var (
	// ErrValidationFailed is returned when input fails validation rules.
	ErrValidationFailed = errors.InvalidArgument("validation failed", nil)

	// ErrPathTraversal is returned when a path escapes the allowed base directory.
	ErrPathTraversal = errors.InvalidArgument("path traversal attempt", nil)
)

// mapValidationError converts go-playground validation failures into
// pkg/errors.InvalidArgument AppErrors. Nil and existing InvalidArgument
// errors are returned unchanged.
func mapValidationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.IsCode(err, errors.CodeInvalidArgument) {
		return err
	}
	if ve, ok := err.(playground.ValidationErrors); ok && len(ve) > 0 {
		return errors.InvalidArgument(fmt.Sprintf("validation failed: %s", ve.Error()), err)
	}
	return errors.InvalidArgument("validation failed", err)
}

// errPathTraversal returns an InvalidArgument error for a path traversal attempt.
func errPathTraversal(targetPath, resolvedPath, baseDir string) error {
	msg := fmt.Sprintf("path traversal attempt: path %s resolves to %s which is not within %s",
		targetPath, resolvedPath, baseDir)
	return errors.InvalidArgument(msg, ErrPathTraversal)
}
