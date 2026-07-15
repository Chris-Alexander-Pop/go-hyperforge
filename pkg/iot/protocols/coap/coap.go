// Package coap provides a stub Constrained Application Protocol (CoAP) client surface.
//
// This is intentionally a protocol stub: types and a Client interface suitable for
// adapters and tests. It does not speak real UDP/DTLS CoAP on the wire yet.
package coap

import (
	"context"
	"time"
)

// Method is a CoAP request method (RFC 7252).
type Method byte

const (
	MethodEmpty  Method = 0
	MethodGET    Method = 1
	MethodPOST   Method = 2
	MethodPUT    Method = 3
	MethodDELETE Method = 4
)

// Code is a CoAP response code class/detail (e.g. 2.05 Content = 69).
type Code byte

const (
	CodeEmpty               Code = 0
	CodeCreated             Code = 65  // 2.01
	CodeDeleted             Code = 66  // 2.02
	CodeValid               Code = 67  // 2.03
	CodeChanged             Code = 68  // 2.04
	CodeContent             Code = 69  // 2.05
	CodeBadRequest          Code = 128 // 4.00
	CodeUnauthorized        Code = 129 // 4.01
	CodeNotFound            Code = 132 // 4.04
	CodeMethodNotAllowed    Code = 133 // 4.05
	CodeInternalServerError Code = 160 // 5.00
)

// MessageType is Confirmable / Non-confirmable / Ack / Reset.
type MessageType byte

const (
	TypeConfirmable     MessageType = 0
	TypeNonConfirmable  MessageType = 1
	TypeAcknowledgement MessageType = 2
	TypeReset           MessageType = 3
)

// Message is a CoAP request or response payload.
type Message struct {
	Type          MessageType
	Code          Code
	MessageID     uint16
	Token         []byte
	Path          string
	Query         string
	Payload       []byte
	ContentFormat int
}

// Request is a client-side CoAP request.
type Request struct {
	Method  Method
	Path    string
	Query   string
	Payload []byte
	Type    MessageType
	Timeout time.Duration
}

// Response is a client-side CoAP response.
type Response struct {
	Code      Code
	Payload   []byte
	MessageID uint16
	Token     []byte
}

// Handler handles an inbound CoAP request (server-side stub).
type Handler func(ctx context.Context, req *Message) (*Message, error)

// Client is the CoAP client interface (stub protocol package).
type Client interface {
	// Connect prepares the transport (no-op for memory stub until UDP lands).
	Connect(ctx context.Context) error
	// Close releases resources.
	Close() error
	// Do sends a request and waits for a response.
	Do(ctx context.Context, req Request) (*Response, error)
	// Get is a convenience for MethodGET.
	Get(ctx context.Context, path string) (*Response, error)
	// Post is a convenience for MethodPOST.
	Post(ctx context.Context, path string, payload []byte) (*Response, error)
	// Observe registers for resource observations (stub; may return Unimplemented).
	Observe(ctx context.Context, path string, handler Handler) error
}

// Config configures a CoAP client.
type Config struct {
	// Address is the peer host:port (UDP), e.g. "192.168.1.10:5683".
	Address string `env:"COAP_ADDRESS"`

	// Timeout is the default request timeout.
	Timeout time.Duration `env:"COAP_TIMEOUT" env-default:"5s"`
}
