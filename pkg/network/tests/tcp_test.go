package network_test

import (
	"context"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network"
	"net"
	"testing"
	"time"
)

func TestTCPServer(t *testing.T) {
	// Find free port
	l, _ := net.Listen("tcp", "localhost:0")
	addr := l.Addr().String()
	l.Close()

	cfg := network.Config{
		Addr:        addr,
		ReadTimeout: 1 * time.Second,
	}

	done := make(chan bool)
	handler := func(conn net.Conn) {
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		if _, err := conn.Write(buf[:n]); err != nil {
			t.Errorf("failed to write echo: %v", err)
		}
		done <- true
	}

	server := network.NewTCPServer(cfg, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	go func() {
		if err := server.ListenAndServe(ctx); err != nil {
			// Context cancellation is expected
			if ctx.Err() == nil {
				t.Logf("server failed: %v", err)
			}
		}
	}()
	time.Sleep(100 * time.Millisecond) // Wait for startup

	// Client
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	msg := "hello"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for server handler")
	}
}
