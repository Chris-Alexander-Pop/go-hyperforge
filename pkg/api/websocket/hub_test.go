package websocket

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHubShutdown_ClosesClients(t *testing.T) {
	hub := NewHub()
	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()

	client := &Client{hub: hub, send: make(chan []byte, 1)}
	hub.register <- client

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients) == 1
	}, time.Second, 10*time.Millisecond)

	hub.Shutdown()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("hub Run did not exit after Shutdown")
	}

	_, ok := <-client.send
	assert.False(t, ok, "client send channel should be closed")

	hub.mu.RLock()
	n := len(hub.clients)
	hub.mu.RUnlock()
	assert.Equal(t, 0, n)

	hub.Shutdown() // idempotent
}

func TestHubBroadcast_DoesNotMutateUnderRLock(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer hub.Shutdown()

	const n = 20
	clients := make([]*Client, n)
	for i := 0; i < n; i++ {
		c := &Client{hub: hub, send: make(chan []byte, 1)}
		clients[i] = c
		hub.register <- c
	}

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients) == n
	}, time.Second, 10*time.Millisecond)

	for _, c := range clients {
		c.send <- []byte("fill")
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		hub.Broadcast <- []byte("overflow")
	}()
	wg.Wait()

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		return len(hub.clients) == 0
	}, 2*time.Second, 20*time.Millisecond)
}

func TestCheckOriginAllowlist(t *testing.T) {
	u := newUpgrader(Config{AllowedOrigins: []string{"https://app.example.com"}})

	reqAllow := httptest.NewRequest(http.MethodGet, "/ws", nil)
	reqAllow.Header.Set("Origin", "https://app.example.com")
	assert.True(t, u.CheckOrigin(reqAllow))

	reqDeny := httptest.NewRequest(http.MethodGet, "/ws", nil)
	reqDeny.Header.Set("Origin", "https://evil.example.com")
	assert.False(t, u.CheckOrigin(reqDeny))

	reqNoOrigin := httptest.NewRequest(http.MethodGet, "/ws", nil)
	assert.True(t, u.CheckOrigin(reqNoOrigin))

	uAll := newUpgrader(Config{AllowedOrigins: []string{"*"}})
	assert.True(t, uAll.CheckOrigin(reqDeny))
}
