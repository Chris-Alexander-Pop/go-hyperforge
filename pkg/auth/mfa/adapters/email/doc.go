// Package email provides an email channel MFA adapter backed by pkg/communication/email.Sender.
//
// This adapter stores enrollments and challenge hashes in memory and delivers OTP
// codes through any email.Sender implementation.
//
// Production path (SendGrid / SES / SMTP):
//
//	import (
//	    sendgrid "github.com/chris-alexander-pop/system-design-library/pkg/communication/email/adapters/sendgrid"
//	    "github.com/chris-alexander-pop/system-design-library/pkg/communication/email"
//	    mfaemail "github.com/chris-alexander-pop/system-design-library/pkg/auth/mfa/adapters/email"
//	)
//
//	sender, err := sendgrid.New(email.Config{SendGridAPIKey: "...", DefaultFrom: "noreply@example.com"})
//	provider, err := mfaemail.New(sender, mfa.Config{
//	    EmailSubject:     "Your verification code",
//	    MessageTemplate:  "Your code is %s",
//	})
//
// Tests: inject communication/email/adapters/memory.
package email
