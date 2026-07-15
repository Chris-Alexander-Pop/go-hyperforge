// Package sms provides an SMS channel MFA adapter backed by pkg/communication/sms.Sender.
//
// This adapter stores enrollments and challenge hashes in memory and delivers OTP
// codes through any sms.Sender implementation.
//
// Production path (Twilio):
//
//	import (
//	    twilio "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms/adapters/twilio"
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms"
//	    mfasms "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/mfa/adapters/sms"
//	)
//
//	sender, err := twilio.New(sms.Config{
//	    TwilioAccountSID: "...",
//	    TwilioAuthToken:  "...",
//	    TwilioFromNumber: "+1...",
//	})
//	provider, err := mfasms.New(sender, mfa.Config{MessageTemplate: "Your code is %s"})
//
// Tests: inject communication/sms/adapters/memory.
package sms
