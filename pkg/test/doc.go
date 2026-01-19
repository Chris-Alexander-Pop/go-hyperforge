/*
Package test provides testing utilities for the system-design-library.

This package includes:
  - Suite: Base test suite with context and testify integration
  - Postgres/Redis helpers for integration testing

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/test"

	type MyTestSuite struct {
		test.Suite
	}

	func (s *MyTestSuite) TestSomething() {
		s.NoError(doSomething(s.Ctx))
	}

	func TestMySuite(t *testing.T) {
		test.Run(t, new(MyTestSuite))
	}
*/
package test
