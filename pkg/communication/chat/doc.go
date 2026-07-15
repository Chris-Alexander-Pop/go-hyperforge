// Package chat provides chat platform integrations for Slack, Discord, and memory.
//
// First-party WebSocket chat rooms are not implemented in this package.
// Wrap senders with NewResilientSender (wired from Config.RetryMax / RetryBackoff)
// and NewInstrumentedSender for observability.
package chat
