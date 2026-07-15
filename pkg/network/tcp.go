package network

import (
	"context"
	"net"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
)

type TCPHandler func(conn net.Conn)

type TCPServer struct {
	cfg     Config
	Handler TCPHandler
}

func NewTCPServer(cfg Config, handler TCPHandler) *TCPServer {
	return &TCPServer{cfg: cfg, Handler: handler}
}

func (s *TCPServer) ListenAndServe(ctx context.Context) error {
	l, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return errors.Wrap(err, "tcp listen")
	}
	defer l.Close()

	logger.L().InfoContext(ctx, "started tcp server", "addr", s.cfg.Addr)

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
			logger.L().ErrorContext(ctx, "tcp accept error", "error", errors.Wrap(err, "tcp accept"))
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			if s.cfg.ReadTimeout > 0 {
				_ = c.SetDeadline(time.Now().Add(s.cfg.ReadTimeout))
			}
			s.Handler(c)
		}(conn)
	}
}
