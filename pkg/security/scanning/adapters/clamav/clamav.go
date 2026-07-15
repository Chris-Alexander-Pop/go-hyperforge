// Package clamav implements scanning.Scanner via the ClamAV daemon protocol.
//
// Scan reads the resource file (Location) and streams it with INSTREAM over TCP.
// Inject Dialer via NewFromDialer for tests; New dials Address with net.Dialer.
package clamav

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning"
)

const (
	defaultChunkSize = 2048
	maxResponseSize  = 4096
)

// Dialer opens a connection to clamd. net.Dialer satisfies this via DialContext.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Config configures the ClamAV scanner adapter.
type Config struct {
	// Address is the clamd TCP address (e.g. "127.0.0.1:3310").
	Address string `env:"CLAMAV_ADDRESS" env-default:"127.0.0.1:3310"`

	// Network is usually "tcp" (default).
	Network string `env:"CLAMAV_NETWORK" env-default:"tcp"`

	// Timeout is the dial/read/write deadline when > 0.
	Timeout time.Duration `env:"CLAMAV_TIMEOUT" env-default:"30s"`

	// ChunkSize is the INSTREAM chunk size (default 2048).
	ChunkSize int `env:"CLAMAV_CHUNK_SIZE"`

	// Dialer overrides the default net.Dialer (tests).
	Dialer Dialer
}

// Scanner implements scanning.Scanner via ClamAV INSTREAM.
type Scanner struct {
	dialer    Dialer
	network   string
	address   string
	timeout   time.Duration
	chunkSize int
	openFile  func(path string) (io.ReadCloser, error)
}

// Ensure Scanner implements scanning.Scanner.
var _ scanning.Scanner = (*Scanner)(nil)

// NewFromDialer wraps an existing Dialer targeting address.
func NewFromDialer(dialer Dialer, address string) (*Scanner, error) {
	if dialer == nil {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "clamav dialer is required", nil)
	}
	if address == "" {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "clamav address is required", nil)
	}
	return &Scanner{
		dialer:    dialer,
		network:   "tcp",
		address:   address,
		chunkSize: defaultChunkSize,
		openFile:  openLocalFile,
	}, nil
}

// New builds a Scanner from Config.
func New(cfg Config) (*Scanner, error) {
	addr := strings.TrimSpace(cfg.Address)
	if addr == "" {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "clamav address is required", nil)
	}
	network := cfg.Network
	if network == "" {
		network = "tcp"
	}
	dialer := cfg.Dialer
	if dialer == nil {
		d := &net.Dialer{}
		if cfg.Timeout > 0 {
			d.Timeout = cfg.Timeout
		}
		dialer = d
	}
	chunk := cfg.ChunkSize
	if chunk <= 0 {
		chunk = defaultChunkSize
	}
	return &Scanner{
		dialer:    dialer,
		network:   network,
		address:   addr,
		timeout:   cfg.Timeout,
		chunkSize: chunk,
		openFile:  openLocalFile,
	}, nil
}

func openLocalFile(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// Scan streams the file at resource.Location to clamd via INSTREAM.
func (s *Scanner) Scan(ctx context.Context, resource scanning.Resource) (*scanning.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if resource.Location == "" {
		return nil, scanning.ErrInvalidResource
	}

	rc, err := s.openFile(resource.Location)
	if err != nil {
		return nil, pkgerrors.New(scanning.CodeInvalidResource, "failed to open scan target", err)
	}
	defer rc.Close()

	conn, err := s.dialer.DialContext(ctx, s.network, s.address)
	if err != nil {
		return nil, pkgerrors.New(scanning.CodeUnavailable, "clamav dial failed", err)
	}
	defer conn.Close()

	if s.timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(s.timeout))
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	threats, err := instream(conn, rc, s.chunkSize)
	if err != nil {
		return nil, pkgerrors.New(scanning.CodeScanFailed, "clamav instream failed", err)
	}

	report := &scanning.Report{
		ResourceID: resource.ID,
		Clean:      len(threats) == 0,
		Threats:    threats,
		ScannedAt:  time.Now().UTC(),
	}
	return report, nil
}

// Ping sends a zPING command and expects "PONG".
func (s *Scanner) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := s.dialer.DialContext(ctx, s.network, s.address)
	if err != nil {
		return pkgerrors.New(scanning.CodeUnavailable, "clamav dial failed", err)
	}
	defer conn.Close()
	if s.timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(s.timeout))
	}
	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		return pkgerrors.New(scanning.CodeUnavailable, "clamav ping write failed", err)
	}
	resp, err := readNullTerminated(conn)
	if err != nil {
		return pkgerrors.New(scanning.CodeUnavailable, "clamav ping read failed", err)
	}
	if strings.TrimSpace(resp) != "PONG" {
		return pkgerrors.New(scanning.CodeUnavailable, fmt.Sprintf("unexpected clamav ping response %q", resp), nil)
	}
	return nil
}

func instream(conn net.Conn, r io.Reader, chunkSize int) ([]string, error) {
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, err
	}

	buf := make([]byte, chunkSize)
	lenBuf := make([]byte, 4)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			binary.BigEndian.PutUint32(lenBuf, uint32(n))
			if _, werr := conn.Write(lenBuf); werr != nil {
				return nil, werr
			}
			if _, werr := conn.Write(buf[:n]); werr != nil {
				return nil, werr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	// End of stream: zero-length chunk.
	binary.BigEndian.PutUint32(lenBuf, 0)
	if _, err := conn.Write(lenBuf); err != nil {
		return nil, err
	}

	resp, err := readNullTerminated(conn)
	if err != nil {
		return nil, err
	}
	return parseInstreamResponse(resp)
}

func readNullTerminated(r io.Reader) (string, error) {
	br := bufio.NewReaderSize(r, maxResponseSize)
	var b strings.Builder
	for {
		c, err := br.ReadByte()
		if err != nil {
			if err == io.EOF && b.Len() > 0 {
				break
			}
			return "", err
		}
		if c == 0 {
			break
		}
		if b.Len() >= maxResponseSize {
			return "", fmt.Errorf("clamav response too large")
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

// parseInstreamResponse parses replies like "stream: OK" or "stream: Name FOUND".
func parseInstreamResponse(resp string) ([]string, error) {
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return nil, fmt.Errorf("empty clamav response")
	}
	// May contain multiple lines for some configurations; take first meaningful line.
	line := resp
	if i := strings.IndexByte(resp, '\n'); i >= 0 {
		line = strings.TrimSpace(resp[:i])
	}
	lower := strings.ToLower(line)
	if strings.HasSuffix(lower, " ok") || lower == "stream: ok" {
		return nil, nil
	}
	if strings.HasSuffix(lower, " found") {
		// "stream: Eicar-Test-Signature FOUND"
		body := line
		if idx := strings.Index(line, ":"); idx >= 0 {
			body = strings.TrimSpace(line[idx+1:])
		}
		body = strings.TrimSpace(strings.TrimSuffix(body, " FOUND"))
		body = strings.TrimSpace(strings.TrimSuffix(body, " Found"))
		body = strings.TrimSpace(strings.TrimSuffix(body, " found"))
		if body == "" {
			body = "unknown"
		}
		return []string{body}, nil
	}
	if strings.HasSuffix(lower, " error") {
		return nil, fmt.Errorf("clamav error: %s", line)
	}
	return nil, fmt.Errorf("unexpected clamav response: %s", line)
}
