package clamav_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/scanning/adapters/clamav"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// mockClamd is a minimal ClamAV daemon for protocol tests.
type mockClamd struct {
	ln       net.Listener
	infected string // substring that triggers FOUND
	mu       sync.Mutex
	lastData []byte
}

func startMockClamd(t *testing.T, infectedMarker string) *mockClamd {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	m := &mockClamd{ln: ln, infected: infectedMarker}
	go m.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return m
}

func (m *mockClamd) Addr() string { return m.ln.Addr().String() }

func (m *mockClamd) serve() {
	for {
		conn, err := m.ln.Accept()
		if err != nil {
			return
		}
		go m.handle(conn)
	}
}

func (m *mockClamd) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	cmd, err := readCmd(conn)
	if err != nil {
		return
	}
	switch cmd {
	case "zPING":
		_, _ = conn.Write([]byte("PONG\x00"))
	case "zINSTREAM":
		m.handleInstream(conn)
	default:
		_, _ = conn.Write([]byte("UNKNOWN COMMAND\x00"))
	}
}

func readCmd(conn net.Conn) (string, error) {
	var b strings.Builder
	buf := make([]byte, 1)
	for {
		if _, err := conn.Read(buf); err != nil {
			return "", err
		}
		if buf[0] == 0 {
			return b.String(), nil
		}
		b.WriteByte(buf[0])
		if b.Len() > 64 {
			return "", errors.New("command too long")
		}
	}
}

func (m *mockClamd) handleInstream(conn net.Conn) {
	var data bytes.Buffer
	lenBuf := make([]byte, 4)
	for {
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}
		size := binary.BigEndian.Uint32(lenBuf)
		if size == 0 {
			break
		}
		chunk := make([]byte, size)
		if _, err := io.ReadFull(conn, chunk); err != nil {
			return
		}
		_, _ = data.Write(chunk)
	}
	m.mu.Lock()
	m.lastData = append([]byte(nil), data.Bytes()...)
	infected := m.infected != "" && bytes.Contains(data.Bytes(), []byte(m.infected))
	m.mu.Unlock()

	if infected {
		_, _ = conn.Write([]byte("stream: Eicar-Test-Signature FOUND\x00"))
		return
	}
	_, _ = conn.Write([]byte("stream: OK\x00"))
}

type dialerFunc func(ctx context.Context, network, address string) (net.Conn, error)

func (f dialerFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

type ClamAVSuite struct {
	test.Suite
	mock *mockClamd
	sc   *clamav.Scanner
	dir  string
}

func (s *ClamAVSuite) SetupTest() {
	s.Suite.SetupTest()
	s.mock = startMockClamd(s.T(), "EICAR")
	var err error
	s.sc, err = clamav.New(clamav.Config{
		Address: s.mock.Addr(),
		Timeout: 5 * time.Second,
	})
	s.Require().NoError(err)
	s.dir = s.T().TempDir()
}

func (s *ClamAVSuite) writeFile(name, content string) string {
	path := filepath.Join(s.dir, name)
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o600))
	return path
}

func (s *ClamAVSuite) TestScanClean() {
	path := s.writeFile("clean.txt", "hello world")
	report, err := s.sc.Scan(s.Ctx, scanning.Resource{ID: "f1", Type: "file", Location: path})
	s.Require().NoError(err)
	s.True(report.Clean)
	s.Empty(report.Threats)
	s.Equal("f1", report.ResourceID)
}

func (s *ClamAVSuite) TestScanInfected() {
	path := s.writeFile("bad.bin", "prefix EICAR suffix")
	report, err := s.sc.Scan(s.Ctx, scanning.Resource{ID: "bad", Type: "file", Location: path})
	s.Require().NoError(err)
	s.False(report.Clean)
	s.Require().Len(report.Threats, 1)
	s.Equal("Eicar-Test-Signature", report.Threats[0])
}

func (s *ClamAVSuite) TestPing() {
	s.Require().NoError(s.sc.Ping(s.Ctx))
}

func (s *ClamAVSuite) TestEmptyLocation() {
	_, err := s.sc.Scan(s.Ctx, scanning.Resource{ID: "x"})
	s.True(errors.Is(err, scanning.ErrInvalidResource))
}

func (s *ClamAVSuite) TestMissingFile() {
	_, err := s.sc.Scan(s.Ctx, scanning.Resource{ID: "x", Location: filepath.Join(s.dir, "nope")})
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, scanning.CodeInvalidResource))
}

func (s *ClamAVSuite) TestDialFailure() {
	sc, err := clamav.NewFromDialer(dialerFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
		return nil, errors.New("refused")
	}), "127.0.0.1:9")
	s.Require().NoError(err)
	path := s.writeFile("a.txt", "x")
	_, err = sc.Scan(s.Ctx, scanning.Resource{ID: "a", Location: path})
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, scanning.CodeUnavailable))
}

func (s *ClamAVSuite) TestNewFromDialerNil() {
	_, err := clamav.NewFromDialer(nil, "127.0.0.1:3310")
	s.Require().Error(err)
	s.True(pkgerrors.IsCode(err, scanning.CodeInvalidResource))
}

func (s *ClamAVSuite) TestNewEmptyAddress() {
	_, err := clamav.New(clamav.Config{})
	s.Require().Error(err)
}

func (s *ClamAVSuite) TestImplementsScanner() {
	var _ scanning.Scanner = s.sc
}

func (s *ClamAVSuite) TestCanceledContext() {
	ctx, cancel := context.WithCancel(s.Ctx)
	cancel()
	path := s.writeFile("c.txt", "x")
	_, err := s.sc.Scan(ctx, scanning.Resource{ID: "c", Location: path})
	s.Require().Error(err)
	s.True(errors.Is(err, context.Canceled))
}

func TestClamAVSuite(t *testing.T) {
	test.Run(t, new(ClamAVSuite))
}
