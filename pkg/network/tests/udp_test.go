package network_test

import (
	"context"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network"
	"net"
	"testing"
	"time"
)

func TestUDPServer(t *testing.T) {
	// Find free port
	conn, _ := net.ListenPacket("udp", "localhost:0")
	addr := conn.LocalAddr().String()
	conn.Close()

	cfg := network.Config{
		Addr: addr,
	}

	done := make(chan string)
	handler := func(addr net.Addr, data []byte) {
		done <- string(data)
	}

	server := network.NewUDPServer(cfg, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.ListenAndServe(ctx); err != nil {
			if ctx.Err() == nil {
				t.Logf("server failed: %v", err)
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)

	// Client
	c, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer c.Close()

	msg := "packet"
	if _, err := c.Write([]byte(msg)); err != nil {
		t.Fatalf("failed to write packet: %v", err)
	}

	select {
	case received := <-done:
		if received != msg {
			t.Errorf("Expected %s, got %s", msg, received)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for udp packet")
	}
}
