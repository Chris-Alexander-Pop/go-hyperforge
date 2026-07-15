package mqtt

import (
	"errors"
	"testing"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type fakeToken struct {
	completed bool
	err       error
}

func (t *fakeToken) Wait() bool                  { return t.completed }
func (t *fakeToken) WaitTimeout(time.Duration) bool { return t.completed }
func (t *fakeToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	if t.completed {
		close(ch)
	}
	return ch
}
func (t *fakeToken) Error() error { return t.err }

func TestWaitToken_Success(t *testing.T) {
	err := waitToken(&fakeToken{completed: true}, time.Second, "publish")
	if err != nil {
		t.Fatal(err)
	}
}

func TestWaitToken_Timeout(t *testing.T) {
	err := waitToken(&fakeToken{completed: false}, time.Millisecond, "connect to MQTT broker")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !pkgerrors.IsCode(err, pkgerrors.CodeDeadlineExceeded) {
		t.Fatalf("code = %s err=%v", pkgerrors.Code(err), err)
	}
}

func TestWaitToken_TokenError(t *testing.T) {
	cause := errors.New("broker refused")
	err := waitToken(&fakeToken{completed: true, err: cause}, time.Second, "publish message")
	if err == nil {
		t.Fatal("expected error")
	}
	if !pkgerrors.IsCode(err, pkgerrors.CodeInternal) {
		t.Fatalf("code = %s", pkgerrors.Code(err))
	}
}

func TestWaitToken_NilToken(t *testing.T) {
	err := waitToken(nil, time.Second, "subscribe")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWaitToken_TimeoutNotSuccess(t *testing.T) {
	// Regression: the old `WaitTimeout && Error` pattern returned nil on timeout.
	tok := &fakeToken{completed: false, err: errors.New("should be ignored")}
	err := waitToken(tok, time.Millisecond, "publish")
	if err == nil {
		t.Fatal("timeout must not be treated as success")
	}
	if !pkgerrors.IsCode(err, pkgerrors.CodeDeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %s", pkgerrors.Code(err))
	}
}
