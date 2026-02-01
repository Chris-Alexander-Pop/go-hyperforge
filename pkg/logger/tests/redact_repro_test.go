package logger_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

func TestRedactHandler_ExtendedKeys(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	r := logger.NewRedactHandler(h)
	l := slog.New(r)

	// Log various sensitive keys that should be redacted
	l.Info("Sensitive data",
		"api_key", "12345-abcde",
		"apikey", "secret-key-value",
		"access_key", "access-key-123",
		"authorization", "Bearer xyz123",
		"cookie", "session_id=abcdef",
		"bearer_token", "token-value", // Should be caught by "token"
		"my_secret", "hidden", // Should be caught by "secret"
	)

	out := buf.String()
	t.Logf("Output: %s", out)

	checks := []struct {
		key      string
		value    string
		redacted bool
	}{
		{"api_key", "12345-abcde", true},
		{"apikey", "secret-key-value", true},
		{"access_key", "access-key-123", true},
		{"authorization", "Bearer xyz123", true},
		{"cookie", "session_id=abcdef", true},
		{"bearer_token", "token-value", true},
		{"my_secret", "hidden", true},
	}

	for _, check := range checks {
		// Check if the original value is present
		if check.redacted {
			if strings.Contains(out, check.value) {
				t.Errorf("Value for key '%s' was LEAKED: %s", check.key, check.value)
			}
			// Check if [REDACTED] is present (not perfect if multiple redacted, but good enough)
			// Ideally we parse the JSON, but string check is faster for repro.
		}
	}

	if !strings.Contains(out, "[REDACTED]") {
		t.Error("No [REDACTED] found in output")
	}
}
