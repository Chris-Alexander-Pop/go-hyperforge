package communication

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// ShouldRetrySend reports whether a send/render error is worth retrying.
// Invalid argument and not-found errors are treated as permanent failures.
func ShouldRetrySend(err error) bool {
	if err == nil {
		return false
	}
	if errors.IsCode(err, errors.CodeInvalidArgument) || errors.IsCode(err, errors.CodeNotFound) {
		return false
	}
	return true
}
