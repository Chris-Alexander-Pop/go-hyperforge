package audit_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type RedactSuite struct {
	*test.Suite
	redactor *audit.Redactor
}

func TestRedactSuite(t *testing.T) {
	s := &RedactSuite{Suite: test.NewSuite()}
	test.Run(t, s)
}

func (s *RedactSuite) SetupTest() {
	s.Suite.SetupTest()
	s.redactor = audit.NewRedactor(audit.DefaultRedactorConfig())
}

func (s *RedactSuite) TestRedactCreditCard() {
	in := "card 4111-1111-1111-1111 used"
	out := s.redactor.Redact(in)
	s.NotContains(out, "4111-1111-1111-1111")
	s.Contains(out, "[REDACTED]")
}

func (s *RedactSuite) TestRedactSSN() {
	out := s.redactor.Redact("ssn 123-45-6789")
	s.NotContains(out, "123-45-6789")
	s.Contains(out, "[REDACTED]")
}

func (s *RedactSuite) TestRedactEmail() {
	out := s.redactor.Redact("contact me@example.com please")
	s.NotContains(out, "me@example.com")
	s.Contains(out, "[REDACTED]")
}

func (s *RedactSuite) TestRedactMapSensitiveFieldName() {
	data := map[string]interface{}{
		"password":      "hunter2",
		"api_key":       "test_key_abcdefghijklmnopqrstuvwxyz",
		"access_token":  "tok_abc",
		"refresh_token": "ref_xyz",
		"authorization": "Bearer abc",
		"user_name":     "alice",
		"nested": map[string]interface{}{
			"secret": "nested-secret",
			"ok":     "visible",
		},
		"tags": []interface{}{"a", "b"},
	}

	out := s.redactor.RedactMap(data)
	s.Equal("[REDACTED]", out["password"])
	s.Equal("[REDACTED]", out["api_key"])
	s.Equal("[REDACTED]", out["access_token"])
	s.Equal("[REDACTED]", out["refresh_token"])
	s.Equal("[REDACTED]", out["authorization"])
	s.Equal("alice", out["user_name"])

	nested, ok := out["nested"].(map[string]interface{})
	s.Require().True(ok)
	s.Equal("[REDACTED]", nested["secret"])
	s.Equal("visible", nested["ok"])

	tags, ok := out["tags"].([]interface{})
	s.Require().True(ok)
	s.Equal([]interface{}{"a", "b"}, tags)
}

func (s *RedactSuite) TestRedactMapPatternInValue() {
	data := map[string]interface{}{
		"note": "email is admin@corp.com",
	}
	out := s.redactor.RedactMap(data)
	s.NotContains(out["note"].(string), "admin@corp.com")
	s.Contains(out["note"].(string), "[REDACTED]")
}

func (s *RedactSuite) TestIsSensitiveField() {
	s.True(audit.IsSensitiveField("password"))
	s.True(audit.IsSensitiveField("UserPassword"))
	s.True(audit.IsSensitiveField("my_api_key"))
	s.True(audit.IsSensitiveField("credit_card_number"))
	s.False(audit.IsSensitiveField("username"))
	s.False(audit.IsSensitiveField("actor_id"))
}

func (s *RedactSuite) TestMaskHelpers() {
	s.Equal("ab***fg", audit.MaskString("abcdefg", 2, 2))
	s.Equal("***", audit.MaskString("abc", 2, 2))
	s.Equal("a***e@example.com", audit.MaskEmail("alice@example.com"))
	s.Equal("*@example.com", audit.MaskEmail("ab@example.com"))
	s.Equal("[INVALID_EMAIL]", audit.MaskEmail("not-an-email"))
	s.Equal("************1111", audit.MaskCreditCard("4111-1111-1111-1111"))
	s.Equal("[INVALID_CC]", audit.MaskCreditCard("12"))
}

func (s *RedactSuite) TestRedactEvent() {
	event := audit.Event{
		ActorIP:        "8.8.8.8",
		Description:    "user me@x.com paid with 4111-1111-1111-1111",
		ErrorMessage:   "token=abc123secretvaluetokenhere",
		ActorUserAgent: "ua",
		Metadata: map[string]interface{}{
			"passwd": "x",
			"count":  3,
		},
	}
	out := s.redactor.RedactEvent(event)
	s.Equal("[REDACTED]", out.Metadata["passwd"])
	s.Equal(3, out.Metadata["count"])
	s.NotContains(out.Description, "me@x.com")
	s.NotContains(out.Description, "4111-1111-1111-1111")
	// ActorIP matches ipv4 pattern
	s.Equal("[REDACTED]", out.ActorIP)
}

func (s *RedactSuite) TestCustomReplacementAndPattern() {
	r := audit.NewRedactor(audit.RedactorConfig{
		Replacement: "***",
		CustomPatterns: map[string]string{
			"ticket": `\bTKT-\d+\b`,
		},
	})
	out := r.Redact("see TKT-99")
	s.Equal("see ***", out)

	data := map[string]interface{}{"secret_key": "value"}
	s.Equal("***", r.RedactMap(data)["secret_key"])
}

func (s *RedactSuite) TestAddPatternError() {
	err := s.redactor.AddPattern("bad", "(", "")
	s.Error(err)
}
