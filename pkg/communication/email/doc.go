// Package email provides email delivery via SendGrid, SES, SMTP, and an in-memory adapter.
//
// Wrap senders with NewResilientSender (wired from Config.RetryMax / RetryBackoff)
// and NewInstrumentedSender for observability.
package email
