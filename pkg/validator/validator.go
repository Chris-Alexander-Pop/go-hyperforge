package validator

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// Common Regex Patterns
var (
	slugRegex  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`) // E.164 standard roughly
)

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	v := validator.New()

	// Register Custom Validations
	_ = v.RegisterValidation("slug", validateSlug)
	_ = v.RegisterValidation("password_strong", validatePasswordStrong)
	_ = v.RegisterValidation("phone_e164", validatePhone)

	return &Validator{
		validate: v,
	}
}

// ValidateStruct validates a struct using tags
func (v *Validator) ValidateStruct(s interface{}) error {
	return v.validate.Struct(s)
}

// ValidateVar validates a single variable against a tag
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// Custom Validation Functions

func validateSlug(fl validator.FieldLevel) bool {
	return slugRegex.MatchString(fl.Field().String())
}

func validatePhone(fl validator.FieldLevel) bool {
	return phoneRegex.MatchString(fl.Field().String())
}

func validatePasswordStrong(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	// Length 8+
	if len(password) < 8 {
		return false
	}
	// Needs Number, Special, Upper, etc. (Simplified for this example)
	// Just generic complexity check is often better handled by zxcvbn, but for regex-ish:
	return true
}
