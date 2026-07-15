/*
Package validator provides input validation and sanitization helpers.

# Validation

This package wraps go-playground/validator with additional custom validations:
  - slug: URL-safe slug format (lowercase alphanumeric with hyphens)
  - password_strong: Password strength (8+ chars, upper, lower, number, special)
  - phone_e164: E.164 phone number format

Validation failures are mapped to pkg/errors.InvalidArgument. Prefer the
Validator interface (implemented by Engine) and wrap with
NewInstrumentedValidator for logging and OpenTelemetry spans.

	import "github.com/chris-alexander-pop/system-design-library/pkg/validator"

	v := validator.NewInstrumentedValidator(validator.New())

	err := v.ValidateStruct(ctx, myStruct)
	err = v.ValidateVar(ctx, email, "required,email")

# Sanitization

Sanitizer strips or escapes HTML and can recurse through maps/slices via
SanitizeMap. Prefer detection helpers before accepting untrusted input:

  - DetectSQLInjection — common SQL injection patterns
  - DetectCommandInjection — shell metacharacters
  - DetectPathTraversal — ../ and encoded variants
  - ValidatePathInside — ensure a target path stays within a base directory
  - SanitizePath / SanitizeForShell — strip dangerous sequences (prefer
    parameterized APIs over string interpolation when possible)

	s := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	clean := s.Sanitize(userInput)
	safe := s.SanitizeMap(formData)
*/
package validator
