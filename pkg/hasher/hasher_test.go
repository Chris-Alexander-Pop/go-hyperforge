package hasher_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/hasher"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type HasherSuite struct {
	*test.Suite
}

func TestHasherSuite(t *testing.T) {
	test.Run(t, &HasherSuite{Suite: test.NewSuite()})
}

func (s *HasherSuite) TestHashAndVerify() {
	h := hasher.New()
	password := "supersecurepassword"

	// 1. Hash the password
	hash, err := h.Hash(password)
	s.NoError(err)
	s.NotEmpty(hash)
	s.Contains(hash, "$")

	// 2. Verify Correct Password
	match := h.Verify(password, hash)
	s.True(match, "Password should match")

	// 3. Verify Incorrect Password
	match = h.Verify("wrongpassword", hash)
	s.False(match, "Wrong password should not match")
}
