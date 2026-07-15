package test_test

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

type exampleSuite struct {
	test.Suite
}

func (s *exampleSuite) TestCtxReady() {
	s.NoError(s.Ctx.Err())
}

func ExampleSuite() {
	// In real tests, call test.Run from a Test* function:
	//   func TestExample(t *testing.T) { test.Run(t, new(exampleSuite)) }
	s := test.NewSuite()
	s.SetupTest()
	fmt.Println(s.Ctx != nil)
	// Output: true
}

func ExampleRun() {
	// Demonstrates the Run entrypoint shape (executed via TestExampleRun).
	fmt.Println("use test.Run(t, new(MySuite))")
	// Output: use test.Run(t, new(MySuite))
}

func TestExampleRun(t *testing.T) {
	test.Run(t, new(exampleSuite))
}
