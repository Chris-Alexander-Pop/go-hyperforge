package email_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa"
	mfaemail "github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/email"
	emailmemory "github.com/chris-alexander-pop/system-design-library/pkg/communication/email/adapters/memory"
)

func TestEmailChannelWithMemorySender(t *testing.T) {
	sender := emailmemory.New()
	mem, ok := sender.(*emailmemory.Sender)
	if !ok {
		t.Fatal("expected concrete memory sender")
	}

	p, err := mfaemail.New(sender, mfa.Config{
		MessageTemplate: "code:%s",
		EmailSubject:    "MFA",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Enroll(context.Background(), "user-1", "user@example.com")
	if err != nil {
		t.Fatal(err)
	}

	msgs := mem.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 email, got %d", len(msgs))
	}
	if msgs[0].Subject != "MFA" {
		t.Fatalf("unexpected subject: %s", msgs[0].Subject)
	}
	code := strings.TrimPrefix(msgs[0].Body.PlainText, "code:")
	if err := p.CompleteEnrollment(context.Background(), "user-1", code); err != nil {
		t.Fatal(err)
	}
}
