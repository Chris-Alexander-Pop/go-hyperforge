// Package push provides push notifications via FCM, APNs, and an in-memory adapter.
//
// WebPush is not implemented yet. Wrap senders with NewResilientSender
// (wired from Config.RetryMax / RetryBackoff) and NewInstrumentedSender.
package push
