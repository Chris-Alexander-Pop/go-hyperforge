package validator

import (
	"context"
	"regexp"
	"unicode"

	playground "github.com/go-playground/validator/v10"
)

// Common Regex Patterns
var (
	slugRegex  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`) // E.164 standard roughly
)

// Validator validates structs and variables against validation tags.
type Validator interface {
	// ValidateStruct validates a struct using validate tags.
	ValidateStruct(ctx context.Context, s interface{}) error

	// ValidateVar validates a single variable against a tag.
	ValidateVar(ctx context.Context, field interface{}, tag string) error
}

// Ensure Engine implements Validator at compile time.
var _ Validator = (*Engine)(nil)

// Engine is the concrete go-playground/validator based implementation.
type Engine struct {
	validate *playground.Validate
}

// New creates a Validator with custom rules registered (slug, password_strong, phone_e164).
func New() *Engine {
	v := playground.New()

	_ = v.RegisterValidation("slug", validateSlug)
	_ = v.RegisterValidation("password_strong", validatePasswordStrong)
	_ = v.RegisterValidation("phone_e164", validatePhone)

	return &Engine{
		validate: v,
	}
}

// ValidateStruct validates a struct using tags. Failures map to errors.InvalidArgument.
func (v *Engine) ValidateStruct(ctx context.Context, s interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return mapValidationError(v.validate.Struct(s))
}

// ValidateVar validates a single variable against a tag. Failures map to errors.InvalidArgument.
func (v *Engine) ValidateVar(ctx context.Context, field interface{}, tag string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return mapValidationError(v.validate.Var(field, tag))
}

// Custom Validation Functions

func validateSlug(fl playground.FieldLevel) bool {
	return slugRegex.MatchString(fl.Field().String())
}

func validatePhone(fl playground.FieldLevel) bool {
	return phoneRegex.MatchString(fl.Field().String())
}

func validatePasswordStrong(fl playground.FieldLevel) bool {
	password := fl.Field().String()

	// Length 8+
	if len(password) < 8 {
		return false
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}
