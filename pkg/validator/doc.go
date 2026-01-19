/*
Package validator provides input validation with custom validation rules.

This package wraps go-playground/validator with additional custom validations:
  - slug: URL-safe slug format
  - password_strong: Password strength validation
  - phone_e164: E.164 phone number format

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/validator"

	v := validator.New()

	// Validate struct
	err := v.ValidateStruct(myStruct)

	// Validate single value
	err := v.ValidateVar(email, "required,email")
*/
package validator
