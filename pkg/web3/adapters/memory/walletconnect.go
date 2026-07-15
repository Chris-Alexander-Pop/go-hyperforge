package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
)

// Ensure WalletConnect implements web3.WalletConnectSession.
var _ web3.WalletConnectSession = (*WalletConnect)(nil)

// WalletConnect is an in-memory WalletConnect session stub.
type WalletConnect struct {
	mu       *concurrency.SmartRWMutex
	sessions map[string]*web3.WCSession
	reqSeq   uint64
}

// NewWalletConnect creates an empty WalletConnect session store.
func NewWalletConnect() *WalletConnect {
	return &WalletConnect{
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "web3-memory-wc"}),
		sessions: make(map[string]*web3.WCSession),
	}
}

// Pair implements WalletConnectSession.
func (w *WalletConnect) Pair(ctx context.Context, uri string) (*web3.WCSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if uri == "" {
		return nil, web3.ErrInvalidConfig("walletconnect uri is required", nil)
	}
	topic, err := randomTopic()
	if err != nil {
		return nil, web3.ErrInvalidConfig("failed to generate topic", err)
	}
	s := &web3.WCSession{
		Topic:    topic,
		URI:      uri,
		Status:   web3.WCStatusPending,
		Metadata: map[string]string{},
	}
	w.mu.Lock()
	w.sessions[topic] = s
	w.mu.Unlock()
	cp := *s
	return &cp, nil
}

// Approve implements WalletConnectSession.
func (w *WalletConnect) Approve(ctx context.Context, topic string, accounts []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	s, ok := w.sessions[topic]
	if !ok {
		return web3.ErrNotFound("walletconnect session", nil)
	}
	s.Status = web3.WCStatusApproved
	s.Accounts = append([]string(nil), accounts...)
	return nil
}

// Reject implements WalletConnectSession.
func (w *WalletConnect) Reject(ctx context.Context, topic string, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	s, ok := w.sessions[topic]
	if !ok {
		return web3.ErrNotFound("walletconnect session", nil)
	}
	s.Status = web3.WCStatusRejected
	if s.Metadata == nil {
		s.Metadata = map[string]string{}
	}
	s.Metadata["reject_reason"] = reason
	return nil
}

// GetSession implements WalletConnectSession.
func (w *WalletConnect) GetSession(ctx context.Context, topic string) (*web3.WCSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	s, ok := w.sessions[topic]
	if !ok {
		return nil, web3.ErrNotFound("walletconnect session", nil)
	}
	cp := *s
	cp.Accounts = append([]string(nil), s.Accounts...)
	if s.Metadata != nil {
		cp.Metadata = make(map[string]string, len(s.Metadata))
		for k, v := range s.Metadata {
			cp.Metadata[k] = v
		}
	}
	return &cp, nil
}

// Disconnect implements WalletConnectSession.
func (w *WalletConnect) Disconnect(ctx context.Context, topic string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	s, ok := w.sessions[topic]
	if !ok {
		return web3.ErrNotFound("walletconnect session", nil)
	}
	s.Status = web3.WCStatusClosed
	return nil
}

// Request implements WalletConnectSession.
func (w *WalletConnect) Request(ctx context.Context, topic, method string, params []byte) (*web3.WCResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	s, ok := w.sessions[topic]
	if !ok {
		return nil, web3.ErrNotFound("walletconnect session", nil)
	}
	if s.Status != web3.WCStatusApproved {
		return &web3.WCResponse{Error: "session not approved"}, nil
	}
	w.reqSeq++
	id := hex.EncodeToString([]byte{byte(w.reqSeq)})
	return &web3.WCResponse{
		ID:     id,
		Result: append([]byte(nil), params...),
	}, nil
}

func randomTopic() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
