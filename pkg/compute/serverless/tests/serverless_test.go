package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/compute/serverless"
	"github.com/chris-alexander-pop/system-design-library/pkg/compute/serverless/adapters/memory"
	"github.com/stretchr/testify/suite"
)

// ServerlessRuntimeSuite tests ServerlessRuntime implementations.
type ServerlessRuntimeSuite struct {
	suite.Suite
	runtime *memory.Runtime
	ctx     context.Context
}

func (s *ServerlessRuntimeSuite) SetupTest() {
	s.runtime = memory.New()
	s.ctx = context.Background()
}

func (s *ServerlessRuntimeSuite) TestCreateAndGetFunction() {
	fn, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{
		Name:    "my-function",
		Runtime: serverless.RuntimeNodeJS18,
		Handler: "index.handler",
	})
	s.Require().NoError(err)
	s.Equal("my-function", fn.Name)
	s.Contains(fn.ARN, "my-function")
	s.Equal("Active", fn.State)

	got, err := s.runtime.GetFunction(s.ctx, "my-function")
	s.Require().NoError(err)
	s.Equal(fn.Name, got.Name)
}

func (s *ServerlessRuntimeSuite) TestCreateDuplicate() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "dup-fn"})
	s.Require().NoError(err)
	_, err = s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "dup-fn"})
	s.Error(err)
}

func (s *ServerlessRuntimeSuite) TestGetNotFound() {
	_, err := s.runtime.GetFunction(s.ctx, "nonexistent")
	s.Error(err)
}

func (s *ServerlessRuntimeSuite) TestListFunctions() {
	for i := 0; i < 3; i++ {
		_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{
			Name: "fn-" + string(rune('a'+i)),
		})
		s.Require().NoError(err)
	}

	functions, err := s.runtime.ListFunctions(s.ctx)
	s.Require().NoError(err)
	s.Len(functions, 3)
}

func (s *ServerlessRuntimeSuite) TestUpdateFunction() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{
		Name:     "update-me",
		MemoryMB: 128,
	})
	s.Require().NoError(err)

	updated, err := s.runtime.UpdateFunction(s.ctx, "update-me", serverless.CreateFunctionOptions{
		MemoryMB: 256,
	})
	s.Require().NoError(err)
	s.Equal(256, updated.MemoryMB)
}

func (s *ServerlessRuntimeSuite) TestDeleteFunction() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "delete-me"})
	s.Require().NoError(err)

	err = s.runtime.DeleteFunction(s.ctx, "delete-me")
	s.Require().NoError(err)

	_, err = s.runtime.GetFunction(s.ctx, "delete-me")
	s.Error(err)
}

func (s *ServerlessRuntimeSuite) TestInvokeSync() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "echo"})
	s.Require().NoError(err)
	s.runtime.RegisterHandler("echo", func(ctx context.Context, payload []byte) ([]byte, error) {
		return append([]byte("response:"), payload...), nil
	})

	result, err := s.runtime.Invoke(s.ctx, serverless.InvokeOptions{
		FunctionName:   "echo",
		Payload:        []byte("hello"),
		InvocationType: serverless.InvocationSync,
	})
	s.Require().NoError(err)
	s.Equal(200, result.StatusCode)
	s.Equal("response:hello", string(result.Payload))
}

func (s *ServerlessRuntimeSuite) TestInvokeSimple() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "simple"})
	s.Require().NoError(err)
	s.runtime.RegisterHandler("simple", func(ctx context.Context, payload []byte) ([]byte, error) {
		return []byte("ok"), nil
	})

	payload, err := s.runtime.InvokeSimple(s.ctx, "simple", []byte("input"))
	s.Require().NoError(err)
	s.Equal("ok", string(payload))
}

func (s *ServerlessRuntimeSuite) TestInvokeAsync() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "async-fn"})
	s.Require().NoError(err)

	result, err := s.runtime.Invoke(s.ctx, serverless.InvokeOptions{
		FunctionName:   "async-fn",
		InvocationType: serverless.InvocationAsync,
	})
	s.Require().NoError(err)
	s.Equal(202, result.StatusCode)
}

func (s *ServerlessRuntimeSuite) TestInvokeDryRun() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "dry-fn"})
	s.Require().NoError(err)

	result, err := s.runtime.Invoke(s.ctx, serverless.InvokeOptions{
		FunctionName:   "dry-fn",
		InvocationType: serverless.InvocationDryRun,
	})
	s.Require().NoError(err)
	s.Equal(204, result.StatusCode)
}

func (s *ServerlessRuntimeSuite) TestInvokeWithError() {
	_, err := s.runtime.CreateFunction(s.ctx, serverless.CreateFunctionOptions{Name: "error-fn"})
	s.Require().NoError(err)
	s.runtime.RegisterHandler("error-fn", func(ctx context.Context, payload []byte) ([]byte, error) {
		return nil, errors.New("something went wrong")
	})

	result, err := s.runtime.Invoke(s.ctx, serverless.InvokeOptions{
		FunctionName:   "error-fn",
		InvocationType: serverless.InvocationSync,
	})
	s.Require().NoError(err)
	s.Equal("Unhandled", result.FunctionError)
}

func (s *ServerlessRuntimeSuite) TestInvokeNotFound() {
	_, err := s.runtime.Invoke(s.ctx, serverless.InvokeOptions{FunctionName: "nonexistent"})
	s.Error(err)
}

func TestServerlessRuntimeSuite(t *testing.T) {
	suite.Run(t, new(ServerlessRuntimeSuite))
}
