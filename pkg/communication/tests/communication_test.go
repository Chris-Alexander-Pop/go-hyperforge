package tests

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication/chat"
	chatmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/chat/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	emailmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/email/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/push"
	pushmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/push/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms"
	smsmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/sms/adapters/memory"
	templatemem "github.com/chris-alexander-pop/system-design-library/pkg/communication/template/adapters/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailMemoryAdapter(t *testing.T) {
	sender := emailmem.New()
	defer sender.Close()

	ctx := context.Background()
	msg := &email.Message{
		From:    "test@example.com",
		To:      []string{"user@example.com"},
		Subject: "Test Email",
		Body:    email.Body{PlainText: "Hello World"},
	}

	err := sender.Send(ctx, msg)
	require.NoError(t, err)

	sent := sender.SentMessages()
	require.Len(t, sent, 1)
	assert.Equal(t, msg, sent[0])
}

func TestSMSMemoryAdapter(t *testing.T) {
	sender := smsmem.New()
	defer sender.Close()

	ctx := context.Background()
	msg := &sms.Message{
		From: "+1234567890",
		To:   "+0987654321",
		Body: "Hello SMS",
	}

	err := sender.Send(ctx, msg)
	require.NoError(t, err)

	sent := sender.SentMessages()
	require.Len(t, sent, 1)
	assert.Equal(t, msg, sent[0])
}

func TestPushMemoryAdapter(t *testing.T) {
	sender := pushmem.New()
	defer sender.Close()

	ctx := context.Background()
	msg := &push.Message{
		Tokens: []string{"token1", "token2"},
		Title:  "Test Push",
		Body:   "Hello Push",
	}

	err := sender.Send(ctx, msg)
	require.NoError(t, err)

	sent := sender.SentMessages()
	require.Len(t, sent, 1)
	assert.Equal(t, msg, sent[0])
}

func TestChatMemoryAdapter(t *testing.T) {
	sender := chatmem.New()
	defer sender.Close()

	ctx := context.Background()
	msg := &chat.Message{
		ChannelID: "general",
		Text:      "Hello Chat",
	}

	err := sender.Send(ctx, msg)
	require.NoError(t, err)

	sent := sender.SentMessages()
	require.Len(t, sent, 1)
	assert.Equal(t, msg, sent[0])
}

func TestTemplateMemoryAdapter(t *testing.T) {
	engine := templatemem.New()
	defer engine.Close()

	engine.AddTemplate("welcome", "Hello {{.Name}}")

	ctx := context.Background()
	result, err := engine.Render(ctx, "welcome", map[string]string{"Name": "World"})
	require.NoError(t, err)
	// The mock implementation uses fmt.Sprintf("%s - %v", content, data)
	assert.Contains(t, result, "Hello {{.Name}}")
	assert.Contains(t, result, "World")
}

func TestInstrumentedWrappers(t *testing.T) {
	// Verify that instrumented wrappers conform to interfaces and don't panic
	t.Run("Email", func(t *testing.T) {
		base := emailmem.New()
		wrapper := email.NewInstrumentedSender(base)
		err := wrapper.Send(context.Background(), &email.Message{To: []string{"test"}})
		require.NoError(t, err)
	})

	t.Run("SMS", func(t *testing.T) {
		base := smsmem.New()
		wrapper := sms.NewInstrumentedSender(base)
		err := wrapper.Send(context.Background(), &sms.Message{To: "123"})
		require.NoError(t, err)
	})
}
