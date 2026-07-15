package validator_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
)

type ValidatorSuite struct {
	*test.Suite
}

func TestValidatorSuite(t *testing.T) {
	test.Run(t, &ValidatorSuite{Suite: test.NewSuite()})
}

type User struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=18"`
}

func (s *ValidatorSuite) TestValidator() {
	v := validator.New()
	ctx := context.Background()

	tests := []struct {
		name    string
		input   User
		wantErr bool
	}{
		{
			name:    "Valid User",
			input:   User{Name: "Alice", Email: "alice@example.com", Age: 25},
			wantErr: false,
		},
		{
			name:    "Missing Name",
			input:   User{Name: "", Email: "alice@example.com", Age: 25},
			wantErr: true,
		},
		{
			name:    "Invalid Email",
			input:   User{Name: "Alice", Email: "not-an-email", Age: 25},
			wantErr: true,
		},
		{
			name:    "Underage",
			input:   User{Name: "Alice", Email: "alice@example.com", Age: 10},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := v.ValidateStruct(ctx, tt.input)
			if tt.wantErr {
				s.Error(err)
				s.True(errors.IsCode(err, errors.CodeInvalidArgument))
			} else {
				s.NoError(err)
			}
		})
	}
}

type SlugInput struct {
	Slug string `validate:"slug"`
}

func (s *ValidatorSuite) TestSlug() {
	v := validator.New()
	ctx := context.Background()

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"ValidSimple", "hello", false},
		{"ValidHyphenated", "hello-world", false},
		{"ValidMulti", "my-cool-slug-123", false},
		{"Uppercase", "Hello", true},
		{"Underscore", "hello_world", true},
		{"LeadingHyphen", "-hello", true},
		{"TrailingHyphen", "hello-", true},
		{"DoubleHyphen", "hello--world", true},
		{"Spaces", "hello world", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := v.ValidateStruct(ctx, SlugInput{Slug: tt.slug})
			if tt.wantErr {
				s.Error(err)
				s.True(errors.IsCode(err, errors.CodeInvalidArgument))
			} else {
				s.NoError(err)
			}
		})
	}
}

type PhoneInput struct {
	Phone string `validate:"phone_e164"`
}

func (s *ValidatorSuite) TestPhoneE164() {
	v := validator.New()
	ctx := context.Background()

	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"ValidUS", "+14155552671", false},
		{"ValidShort", "+12125551212", false},
		{"MissingPlus", "14155552671", true},
		{"LeadingZero", "+014155552671", true},
		{"Letters", "+1abc5552671", true},
		{"Empty", "", true},
		{"TooLong", "+1234567890123456", true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := v.ValidateStruct(ctx, PhoneInput{Phone: tt.phone})
			if tt.wantErr {
				s.Error(err)
				s.True(errors.IsCode(err, errors.CodeInvalidArgument))
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ValidatorSuite) TestValidateVar() {
	v := validator.New()
	ctx := context.Background()

	s.NoError(v.ValidateVar(ctx, "alice@example.com", "required,email"))
	err := v.ValidateVar(ctx, "not-an-email", "required,email")
	s.Error(err)
	s.True(errors.IsCode(err, errors.CodeInvalidArgument))

	s.NoError(v.ValidateVar(ctx, "hello-world", "slug"))
	s.Error(v.ValidateVar(ctx, "Hello_World", "slug"))

	s.NoError(v.ValidateVar(ctx, "+14155552671", "phone_e164"))
	s.Error(v.ValidateVar(ctx, "4155552671", "phone_e164"))
}

func (s *ValidatorSuite) TestInstrumentedValidator() {
	v := validator.NewInstrumentedValidator(validator.New())
	ctx := context.Background()

	s.NoError(v.ValidateStruct(ctx, User{Name: "Alice", Email: "alice@example.com", Age: 25}))
	err := v.ValidateVar(ctx, "bad", "email")
	s.Error(err)
	s.True(errors.IsCode(err, errors.CodeInvalidArgument))
}
