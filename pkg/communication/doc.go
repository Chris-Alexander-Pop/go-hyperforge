/*
Package communication provides messaging and notification services.

Subpackages:

  - chat: Real-time chat (WebSocket, Slack, Discord)
  - email: Email delivery (SendGrid, SES, Mailgun)
  - push: Push notifications (FCM, APNs, WebPush)
  - sms: SMS messaging (Twilio, SNS)
  - template: Message templating

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/communication/email"

	sender, err := sendgrid.New(cfg)
	err := sender.Send(ctx, email.Message{To: "user@example.com", Subject: "Hello"})
*/
package communication
