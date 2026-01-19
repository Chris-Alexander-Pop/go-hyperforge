package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Suite wraps testify's suite with additional helper methods for this project
type Suite struct {
	suite.Suite
	Ctx context.Context
}

// SetupTest is called before each test in the suite
func (s *Suite) SetupTest() {
	s.Ctx = context.Background()
}

// NewSuite creates a new test suite
func NewSuite() *Suite {
	return &Suite{}
}

// Assert is a helper to access assertions directly if needed (though s.Equal(...) works too)
func (s *Suite) Assert() *assert.Assertions {
	return s.Assertions
}

// Run is a helper function to run a suite from a standard Test* function
func Run(t *testing.T, s suite.TestingSuite) {
	suite.Run(t, s)
}
