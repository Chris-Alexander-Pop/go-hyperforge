// Package network provides TCP and UDP server implementations.
//
// These are low-level network primitives for building custom protocols.
// For HTTP/REST APIs, use pkg/api instead.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/network"
//
//	server := network.NewTCPServer(":9000", handler)
//	server.ListenAndServe(ctx)
package network

import (
	"context"
	"net"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

type TCPHandler func(conn net.Conn)

type TCPServer struct {
	Addr    string
	Handler TCPHandler
}

func NewTCPServer(addr string, handler TCPHandler) *TCPServer {
	return &TCPServer{Addr: addr, Handler: handler}
}

func (s *TCPServer) ListenAndServe(ctx context.Context) error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer l.Close()

	logger.L().InfoContext(ctx, "started tcp server", "addr", s.Addr)

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil // shutdown
			}
			logger.L().ErrorContext(ctx, "tcp accept error", "error", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			s.Handler(c)
		}(conn)
	}
}
