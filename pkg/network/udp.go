package network

import (
	"context"
	"net"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
)

type UDPHandler func(addr net.Addr, data []byte)

type UDPServer struct {
	cfg        Config
	Handler    UDPHandler
	BufferSize int
}

func NewUDPServer(cfg Config, handler UDPHandler) *UDPServer {
	return &UDPServer{
		cfg:        cfg,
		Handler:    handler,
		BufferSize: 4096, // Tunable
	}
}

func (s *UDPServer) ListenAndServe(ctx context.Context) error {
	pc, err := net.ListenPacket("udp", s.cfg.Addr)
	if err != nil {
		return errors.Wrap(err, "udp listen")
	}
	defer pc.Close()

	logger.L().InfoContext(ctx, "started udp server", "addr", s.cfg.Addr)

	go func() {
		<-ctx.Done()
		pc.Close()
	}()

	buf := make([]byte, s.BufferSize)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			logger.L().ErrorContext(ctx, "udp read error", "error", errors.Wrap(err, "udp read"))
			continue
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		go s.Handler(addr, data)
	}
}
