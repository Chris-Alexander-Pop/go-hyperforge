/*
Package test provides testing utilities for Hyperforge packages.

This package includes:
  - Suite: Base test suite with context and testify integration
  - StartPostgres / StartRedis: optional testcontainers helpers (skipped in -short;
    terminate via t.Cleanup). Prefer memory adapters for unit tests.

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/test"

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
