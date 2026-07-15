package gateway

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// ErrAllProvidersFailed is returned when every configured provider fails.
var ErrAllProvidersFailed = errors.Unavailable("all llm gateway providers failed", nil)
