package tests

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/communication"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/chat"
	chatmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/chat/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
	emailmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/email/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email/adapters/sendgrid"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/email/adapters/smtp"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/push"
	pushmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/push/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms"
	smsmem "github.com/chris-alexander-pop/system-design-library/pkg/communication/sms/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/communication/sms/adapters/twilio"
	tmplhtml "github.com/chris-alexander-pop/system-design-library/pkg/communication/template/adapters/html"
	templatemem "github.com/chris-alexander-pop/system-design-library/pkg/communication/template/adapters/memory"
	tmpltext "github.com/chris-alexander-pop/system-design-library/pkg/communication/template/adapters/text"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
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

	sent := sender.(*emailmem.Sender).SentMessages()
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

	sent := sender.(*smsmem.Sender).SentMessages()
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

	sent := sender.(*pushmem.Sender).SentMessages()
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

	sent := sender.(*chatmem.Sender).SentMessages()
	require.Len(t, sent, 1)
	assert.Equal(t, msg, sent[0])
}

func TestTemplateMemoryAdapter(t *testing.T) {
	engine := templatemem.New()
	defer engine.Close()

	require.NoError(t, engine.AddTemplate("welcome", "Hello {{.Name}}"))

	ctx := context.Background()
	result, err := engine.Render(ctx, "welcome", map[string]string{"Name": "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result)
}

func TestTemplateTextAndHTMLAdapters(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		engine, err := tmpltext.NewFromString("greet", "Hi {{.Name}}")
		require.NoError(t, err)
		defer engine.Close()

		out, err := engine.Render(context.Background(), "greet", map[string]string{"Name": "Ada"})
		require.NoError(t, err)
		assert.Equal(t, "Hi Ada", out)

		_, err = engine.Render(context.Background(), "missing", nil)
		require.Error(t, err)
		assert.True(t, communication.IsNotFound(err))
	})

	t.Run("html escapes", func(t *testing.T) {
		engine, err := tmplhtml.NewFromString("page", "<p>{{.Name}}</p>")
		require.NoError(t, err)
		defer engine.Close()

		out, err := engine.Render(context.Background(), "page", map[string]string{"Name": "<script>"})
		require.NoError(t, err)
		assert.Equal(t, "<p>&lt;script&gt;</p>", out)
	})
}

func TestBuildMIME(t *testing.T) {
	raw, err := email.BuildMIME(&email.Message{
		From:    "a@example.com",
		To:      []string{"b@example.com"},
		Subject: "Subj",
		Body:    email.Body{PlainText: "hello", HTML: "<b>hello</b>"},
		Attachments: []email.Attachment{{
			Filename:    "note.txt",
			Content:     []byte("note"),
			ContentType: "text/plain",
		}},
	})
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "Subject: Subj")
	assert.Contains(t, s, "multipart/mixed")
	assert.Contains(t, s, "note.txt")
}

func TestResilientEmailRetries(t *testing.T) {
	var attempts atomic.Int32
	base := &flakyEmailSender{failTimes: 2, attempts: &attempts}
	cfg := email.ResilientConfigFrom(email.Config{RetryMax: 3, RetryBackoff: time.Millisecond})
	sender := email.NewResilientSender(base, cfg)

	err := sender.Send(context.Background(), &email.Message{To: []string{"a@b.c"}, Subject: "x"})
	require.NoError(t, err)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestResilientEmailSkipsInvalidArgument(t *testing.T) {
	var attempts atomic.Int32
	base := &flakyEmailSender{permanent: pkgerrors.InvalidArgument("bad", nil), attempts: &attempts}
	sender := email.NewResilientSender(base, email.ResilientConfig{
		RetryEnabled:     true,
		RetryMaxAttempts: 5,
		RetryBackoff:     time.Millisecond,
	})

	err := sender.Send(context.Background(), &email.Message{To: []string{"a@b.c"}})
	require.Error(t, err)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestResilientConfigFromChannels(t *testing.T) {
	assert.Equal(t, 3, sms.ResilientConfigFrom(sms.Config{RetryMax: 3}).RetryMaxAttempts)
	assert.Equal(t, 2, push.ResilientConfigFrom(push.Config{RetryMax: 2}).RetryMaxAttempts)
	assert.Equal(t, 4, chat.ResilientConfigFrom(chat.Config{RetryMax: 4}).RetryMaxAttempts)
}

func TestAdapterValidation(t *testing.T) {
	_, err := sendgrid.New(email.Config{Driver: communication.DriverSendGrid})
	require.Error(t, err)

	_, err = smtp.New(email.Config{Driver: communication.DriverSMTP})
	require.Error(t, err)

	_, err = twilio.New(sms.Config{Driver: communication.DriverTwilio})
	require.Error(t, err)
}

func TestDriverConstants(t *testing.T) {
	assert.Equal(t, "sendgrid", communication.DriverSendGrid)
	assert.Equal(t, "twilio", communication.DriverTwilio)
	assert.Equal(t, "fcm", communication.DriverFCM)
	assert.Equal(t, "slack", communication.DriverSlack)
	assert.Equal(t, "html", communication.DriverHTML)
}

func TestShouldRetrySend(t *testing.T) {
	assert.False(t, communication.ShouldRetrySend(nil))
	assert.False(t, communication.ShouldRetrySend(pkgerrors.InvalidArgument("x", nil)))
	assert.False(t, communication.ShouldRetrySend(communication.ErrTemplateNotFound))
	assert.True(t, communication.ShouldRetrySend(pkgerrors.Internal("boom", nil)))
}

func TestInstrumentedWrappers(t *testing.T) {
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

	t.Run("ResilientChat", func(t *testing.T) {
		base := chatmem.New()
		wrapper := chat.NewResilientSender(base, chat.ResilientConfigFrom(chat.Config{RetryMax: 2, RetryBackoff: time.Millisecond}))
		err := wrapper.Send(context.Background(), &chat.Message{ChannelID: "c", Text: "hi"})
		require.NoError(t, err)
	})
}

type flakyEmailSender struct {
	failTimes int32
	attempts  *atomic.Int32
	permanent error
}

func (f *flakyEmailSender) Send(ctx context.Context, msg *email.Message) error {
	n := f.attempts.Add(1)
	if f.permanent != nil {
		return f.permanent
	}
	if n <= f.failTimes {
		return pkgerrors.Internal("transient", errors.New("fail"))
	}
	return nil
}

func (f *flakyEmailSender) SendBatch(ctx context.Context, msgs []*email.Message) error {
	for _, m := range msgs {
		if err := f.Send(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

func (f *flakyEmailSender) Close() error { return nil }
