package sms_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/mfa"
	mfasms "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/mfa/adapters/sms"
	smsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms/adapters/memory"
)

func TestSMSChannelWithMemorySender(t *testing.T) {
	sender := smsmemory.New()
	mem, ok := sender.(*smsmemory.Sender)
	if !ok {
		t.Fatal("expected concrete memory sender")
	}

	p, err := mfasms.New(sender, mfa.Config{MessageTemplate: "code:%s"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Enroll(context.Background(), "user-1", "+15550001111")
	if err != nil {
		t.Fatal(err)
	}

	msgs := mem.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 SMS, got %d", len(msgs))
	}
	if msgs[0].To != "+15550001111" {
		t.Fatalf("unexpected destination: %s", msgs[0].To)
	}
	code := strings.TrimPrefix(msgs[0].Body, "code:")
	if err := p.CompleteEnrollment(context.Background(), "user-1", code); err != nil {
		t.Fatal(err)
	}

	if err := p.SendChallenge(context.Background(), "user-1"); err != nil {
		t.Fatal(err)
	}
	msgs = mem.SentMessages()
	code = strings.TrimPrefix(msgs[len(msgs)-1].Body, "code:")
	okVerify, err := p.Verify(context.Background(), "user-1", code)
	if err != nil || !okVerify {
		t.Fatalf("verify: ok=%v err=%v", okVerify, err)
	}
}
