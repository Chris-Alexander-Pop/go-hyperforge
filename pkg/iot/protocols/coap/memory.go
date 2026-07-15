package coap

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
)

// Memory is an in-process CoAP Client stub for tests (no UDP).
type Memory struct {
	mu        *concurrency.SmartRWMutex
	handlers  map[string]Handler
	connected bool
	cfg       Config
}

// NewMemory creates a memory-backed CoAP client stub.
func NewMemory(cfg Config) *Memory {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 0 // Do uses context deadline when set
	}
	return &Memory{
		handlers: make(map[string]Handler),
		cfg:      cfg,
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "coap-memory"}),
	}
}

var _ Client = (*Memory)(nil)

// RegisterHandler installs a path handler for Do/Get/Post (server side in-process).
func (m *Memory) RegisterHandler(path string, h Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[path] = h
}

func (m *Memory) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *Memory) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *Memory) Do(ctx context.Context, req Request) (*Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	connected := m.connected
	h := m.handlers[req.Path]
	m.mu.RUnlock()
	if !connected {
		return nil, iot.ErrNotConnected()
	}
	if h == nil {
		return &Response{Code: CodeNotFound}, nil
	}
	msg := &Message{
		Type:    req.Type,
		Path:    req.Path,
		Query:   req.Query,
		Payload: req.Payload,
	}
	switch req.Method {
	case MethodGET:
		msg.Code = Code(MethodGET)
	case MethodPOST:
		msg.Code = Code(MethodPOST)
	case MethodPUT:
		msg.Code = Code(MethodPUT)
	case MethodDELETE:
		msg.Code = Code(MethodDELETE)
	}
	out, err := h(ctx, msg)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return &Response{Code: CodeEmpty}, nil
	}
	return &Response{
		Code:      out.Code,
		Payload:   out.Payload,
		MessageID: out.MessageID,
		Token:     out.Token,
	}, nil
}

func (m *Memory) Get(ctx context.Context, path string) (*Response, error) {
	return m.Do(ctx, Request{Method: MethodGET, Path: path, Type: TypeConfirmable})
}

func (m *Memory) Post(ctx context.Context, path string, payload []byte) (*Response, error) {
	return m.Do(ctx, Request{Method: MethodPOST, Path: path, Payload: payload, Type: TypeConfirmable})
}

func (m *Memory) Observe(ctx context.Context, path string, handler Handler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.Unimplemented("coap observe not implemented in stub", nil)
}
