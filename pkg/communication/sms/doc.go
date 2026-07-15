// Package sms provides SMS delivery via Twilio, SNS, and an in-memory adapter.
//
// Wrap senders with NewResilientSender (wired from Config.RetryMax / RetryBackoff)
// and NewInstrumentedSender for observability.
package sms
