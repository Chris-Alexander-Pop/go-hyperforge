package channel

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

type recordingDeliverer struct {
	mu       *concurrency.SmartRWMutex
	messages []struct {
		destination string
		body        string
	}
}

func newRecordingDeliverer() *recordingDeliverer {
	return &recordingDeliverer{
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "test-deliverer"}),
	}
}

func (d *recordingDeliverer) Deliver(ctx context.Context, destination, body string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = append(d.messages, struct {
		destination string
		body        string
	}{destination, body})
	return nil
}

func (d *recordingDeliverer) lastCode() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.messages) == 0 {
		return ""
	}
	body := d.messages[len(d.messages)-1].body
	// Template: "Your verification code is %s"
	const prefix = "Your verification code is "
	if len(body) > len(prefix) {
		return body[len(prefix):]
	}
	return body
}

func TestChannelEnrollmentAndVerify(t *testing.T) {
	d := newRecordingDeliverer()
	p, err := New(d, "sms", mfa.Config{
		CodeDigits:      6,
		MessageTemplate: "Your verification code is %s",
	})
	if err != nil {
		t.Fatal(err)
	}

	codes, err := p.Enroll(context.Background(), "u1", "+15551212")
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) == 0 {
		t.Fatal("expected recovery codes")
	}

	enrollCode := d.lastCode()
	if enrollCode == "" {
		t.Fatal("expected delivered enrollment code")
	}
	if err := p.CompleteEnrollment(context.Background(), "u1", enrollCode); err != nil {
		t.Fatal(err)
	}

	if err := p.SendChallenge(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	ok, err := p.Verify(context.Background(), "u1", d.lastCode())
	if err != nil || !ok {
		t.Fatalf("verify failed: ok=%v err=%v", ok, err)
	}

	// Code is single-use.
	ok, err = p.Verify(context.Background(), "u1", d.lastCode())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected reused code to fail")
	}
}

func TestRecoverAndDisable(t *testing.T) {
	d := newRecordingDeliverer()
	p, err := New(d, "email", mfa.Config{MessageTemplate: "Your verification code is %s"})
	if err != nil {
		t.Fatal(err)
	}

	recovery, err := p.Enroll(context.Background(), "u2", "u2@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.CompleteEnrollment(context.Background(), "u2", d.lastCode()); err != nil {
		t.Fatal(err)
	}

	ok, err := p.Recover(context.Background(), "u2", recovery[0])
	if err != nil || !ok {
		t.Fatalf("recover failed: ok=%v err=%v", ok, err)
	}

	if err := p.Disable(context.Background(), "u2"); err != nil {
		t.Fatal(err)
	}
	if err := p.Disable(context.Background(), "u2"); err == nil {
		t.Fatal("expected not found on second disable")
	}
}
