package validator_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

type PasswordSuite struct {
	*test.Suite
}

func TestPasswordSuite(t *testing.T) {
	test.Run(t, &PasswordSuite{Suite: test.NewSuite()})
}

type UserPassword struct {
	Password string `validate:"password_strong"`
}

func (s *PasswordSuite) TestStrongPassword() {
	v := validator.New()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"TooShort", "Pass1!", true},
		{"NoUpper", "password123!", true},
		{"NoLower", "PASSWORD123!", true},
		{"NoNumber", "Password!", true},
		{"NoSpecial", "Password123", true},
		{"Valid", "Password123!", false},
		{"ValidComplex", "Str0ng@P4ssw0rd", false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := v.ValidateStruct(UserPassword{Password: tt.password})
			if tt.wantErr {
				s.Error(err, "expected error for password: %s", tt.password)
			} else {
				s.NoError(err, "expected no error for password: %s", tt.password)
			}
		})
	}
}
