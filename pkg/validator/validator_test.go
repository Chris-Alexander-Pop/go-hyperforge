package validator_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
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
			err := v.ValidateStruct(tt.input)
			if tt.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}
