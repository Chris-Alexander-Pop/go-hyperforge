/*
Package communication provides messaging and notification services.

Subpackages:

  - chat: Chat platform integrations (Slack, Discord; memory for tests)
  - email: Email delivery (SendGrid, SES, SMTP; memory for tests)
  - push: Push notifications (FCM, APNs; memory for tests)
  - sms: SMS messaging (Twilio, SNS; memory for tests)
  - template: Message templating (memory, text/template, html/template)

Planned but not yet implemented: Mailgun, WebPush, and first-party WebSocket chat.

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email"
	import "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email/adapters/sendgrid"

	sender, err := sendgrid.New(cfg)
	err = sender.Send(ctx, &email.Message{To: []string{"user@example.com"}, Subject: "Hello"})
*/
package communication
