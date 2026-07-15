package coap

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
)

// UDP is a datagram CoAP client/server over UDP.
//
// Framing is a minimal RFC 7252 subset: 4-byte header + token + Uri-Path options
// + 0xFF payload marker. Enough for listen/exchange tests without a full stack.
type UDP struct {
	mu        sync.RWMutex
	conn      net.PacketConn
	remote    *net.UDPAddr
	handlers  map[string]Handler
	connected atomic.Bool
	closed    atomic.Bool
	msgID     atomic.Uint32
	cfg       Config
	serveWG   sync.WaitGroup
	stopCh    chan struct{}
}

// NewUDP creates a UDP CoAP peer. Address is the remote host:port for client Do,
// or the local bind address when Listen is used (e.g. ":5683").
func NewUDP(cfg Config) *UDP {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &UDP{
		handlers: make(map[string]Handler),
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}
}

var _ Client = (*UDP)(nil)

// RegisterHandler installs a path handler for inbound requests (server mode).
func (u *UDP) RegisterHandler(path string, h Handler) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.handlers[path] = h
}

// Listen binds a UDP socket and starts serving inbound requests.
// Address defaults to cfg.Address, or ":0" when empty.
func (u *UDP) Listen(ctx context.Context, address string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if address == "" {
		address = u.cfg.Address
	}
	if address == "" {
		address = ":0"
	}
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return iot.ErrConnectionFailed(err)
	}
	u.mu.Lock()
	u.conn = pc
	u.mu.Unlock()
	u.connected.Store(true)

	u.serveWG.Add(1)
	go u.serveLoop()
	return nil
}

// LocalAddr returns the bound UDP address (after Listen or Connect).
func (u *UDP) LocalAddr() net.Addr {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if u.conn == nil {
		return nil
	}
	return u.conn.LocalAddr()
}

// Connect dials a remote UDP peer (client mode). Does not start a serve loop
// unless Listen was already called; inbound replies are read in Do.
func (u *UDP) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if u.cfg.Address == "" {
		return iot.ErrInvalidConfig("coap address is required", nil)
	}
	raddr, err := net.ResolveUDPAddr("udp", u.cfg.Address)
	if err != nil {
		return iot.ErrConnectionFailed(err)
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.conn == nil {
		pc, err := net.ListenPacket("udp", ":0")
		if err != nil {
			return iot.ErrConnectionFailed(err)
		}
		u.conn = pc
	}
	u.remote = raddr
	u.connected.Store(true)
	return nil
}

// Close stops the serve loop and closes the socket.
func (u *UDP) Close() error {
	if u.closed.Swap(true) {
		return nil
	}
	u.connected.Store(false)
	select {
	case <-u.stopCh:
	default:
		close(u.stopCh)
	}
	u.mu.Lock()
	conn := u.conn
	u.conn = nil
	u.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	u.serveWG.Wait()
	return nil
}

func (u *UDP) serveLoop() {
	defer u.serveWG.Done()
	buf := make([]byte, 2048)
	for {
		select {
		case <-u.stopCh:
			return
		default:
		}
		u.mu.RLock()
		conn := u.conn
		u.mu.RUnlock()
		if conn == nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if u.closed.Load() {
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) || err == io.EOF {
				return
			}
			continue
		}
		req, err := decodeMessage(buf[:n])
		if err != nil || req == nil {
			continue
		}
		u.mu.RLock()
		h := u.handlers[req.Path]
		u.mu.RUnlock()
		var resp *Message
		if h == nil {
			resp = &Message{
				Type:      TypeAcknowledgement,
				Code:      CodeNotFound,
				MessageID: req.MessageID,
				Token:     append([]byte(nil), req.Token...),
				Path:      req.Path,
			}
		} else {
			out, herr := h(context.Background(), req)
			if herr != nil || out == nil {
				resp = &Message{
					Type:      TypeAcknowledgement,
					Code:      CodeInternalServerError,
					MessageID: req.MessageID,
					Token:     append([]byte(nil), req.Token...),
				}
			} else {
				resp = out
				resp.Type = TypeAcknowledgement
				resp.MessageID = req.MessageID
				if len(resp.Token) == 0 {
					resp.Token = append([]byte(nil), req.Token...)
				}
			}
		}
		raw, err := encodeMessage(resp)
		if err != nil {
			continue
		}
		_, _ = conn.WriteTo(raw, addr)
	}
}

// Do sends a request to the configured remote and waits for a matching response.
func (u *UDP) Do(ctx context.Context, req Request) (*Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !u.connected.Load() {
		return nil, iot.ErrNotConnected()
	}
	u.mu.RLock()
	conn := u.conn
	remote := u.remote
	u.mu.RUnlock()
	if conn == nil || remote == nil {
		return nil, iot.ErrNotConnected()
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = u.cfg.Timeout
	}
	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}

	id := uint16(u.msgID.Add(1))
	token := []byte{byte(id >> 8), byte(id)}
	msg := &Message{
		Type:      req.Type,
		Code:      Code(req.Method),
		MessageID: id,
		Token:     token,
		Path:      req.Path,
		Query:     req.Query,
		Payload:   req.Payload,
	}
	if msg.Type == 0 {
		msg.Type = TypeConfirmable
	}
	raw, err := encodeMessage(msg)
	if err != nil {
		return nil, iot.ErrInvalidConfig("encode coap message", err)
	}
	if _, err := conn.WriteTo(raw, remote); err != nil {
		return nil, iot.ErrConnectionFailed(err)
	}

	buf := make([]byte, 2048)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		_ = conn.SetReadDeadline(deadline)
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return nil, iot.ErrTimeout("coap exchange", err)
			}
			return nil, iot.ErrConnectionFailed(err)
		}
		respMsg, err := decodeMessage(buf[:n])
		if err != nil || respMsg == nil {
			continue
		}
		if respMsg.MessageID != id {
			continue
		}
		return &Response{
			Code:      respMsg.Code,
			Payload:   respMsg.Payload,
			MessageID: respMsg.MessageID,
			Token:     respMsg.Token,
		}, nil
	}
}

// Get is a convenience for MethodGET.
func (u *UDP) Get(ctx context.Context, path string) (*Response, error) {
	return u.Do(ctx, Request{Method: MethodGET, Path: path, Type: TypeConfirmable})
}

// Post is a convenience for MethodPOST.
func (u *UDP) Post(ctx context.Context, path string, payload []byte) (*Response, error) {
	return u.Do(ctx, Request{Method: MethodPOST, Path: path, Payload: payload, Type: TypeConfirmable})
}

// Observe is not implemented on the UDP stub.
func (u *UDP) Observe(ctx context.Context, path string, handler Handler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.Unimplemented("coap observe not implemented", nil)
}

// --- minimal CoAP encode/decode ---

const (
	optUriPath  = 11
	optUriQuery = 15
	payloadMark = 0xff
)

func encodeMessage(m *Message) ([]byte, error) {
	if m == nil {
		return nil, errors.InvalidArgument("nil coap message", nil)
	}
	tkl := len(m.Token)
	if tkl > 8 {
		return nil, errors.InvalidArgument("token too long", nil)
	}
	hdr := make([]byte, 4)
	// Ver=1 | Type | TKL
	hdr[0] = (1 << 6) | (byte(m.Type&0x3) << 4) | byte(tkl&0xf)
	hdr[1] = byte(m.Code)
	binary.BigEndian.PutUint16(hdr[2:], m.MessageID)

	out := append(hdr, m.Token...)
	var lastOpt uint16
	if m.Path != "" {
		for _, seg := range splitPath(m.Path) {
			out = appendOption(out, &lastOpt, optUriPath, []byte(seg))
		}
	}
	if m.Query != "" {
		out = appendOption(out, &lastOpt, optUriQuery, []byte(m.Query))
	}
	if len(m.Payload) > 0 {
		out = append(out, payloadMark)
		out = append(out, m.Payload...)
	}
	return out, nil
}

func appendOption(buf []byte, last *uint16, num uint16, val []byte) []byte {
	delta := num - *last
	*last = num
	lenV := len(val)
	var first byte
	var ext []byte
	switch {
	case delta < 13:
		first = byte(delta << 4)
	case delta < 269:
		first = 13 << 4
		ext = append(ext, byte(delta-13))
	default:
		first = 14 << 4
		d := delta - 269
		ext = append(ext, byte(d>>8), byte(d))
	}
	switch {
	case lenV < 13:
		first |= byte(lenV)
	case lenV < 269:
		first |= 13
		ext = append(ext, byte(lenV-13))
	default:
		first |= 14
		l := lenV - 269
		ext = append(ext, byte(l>>8), byte(l))
	}
	buf = append(buf, first)
	buf = append(buf, ext...)
	buf = append(buf, val...)
	return buf
}

func decodeMessage(data []byte) (*Message, error) {
	if len(data) < 4 {
		return nil, errors.InvalidArgument("coap datagram too short", nil)
	}
	ver := data[0] >> 6
	if ver != 1 {
		return nil, errors.InvalidArgument("unsupported coap version", nil)
	}
	typ := MessageType((data[0] >> 4) & 0x3)
	tkl := int(data[0] & 0xf)
	if tkl > 8 || len(data) < 4+tkl {
		return nil, errors.InvalidArgument("invalid token length", nil)
	}
	m := &Message{
		Type:      typ,
		Code:      Code(data[1]),
		MessageID: binary.BigEndian.Uint16(data[2:4]),
		Token:     append([]byte(nil), data[4:4+tkl]...),
	}
	i := 4 + tkl
	var lastOpt uint16
	var pathSegs []string
	for i < len(data) {
		if data[i] == payloadMark {
			i++
			m.Payload = append([]byte(nil), data[i:]...)
			break
		}
		first := data[i]
		i++
		delta := uint16(first >> 4)
		olen := int(first & 0xf)
		if delta == 13 {
			if i >= len(data) {
				return nil, errors.InvalidArgument("truncated option delta", nil)
			}
			delta = uint16(data[i]) + 13
			i++
		} else if delta == 14 {
			if i+1 >= len(data) {
				return nil, errors.InvalidArgument("truncated option delta", nil)
			}
			delta = binary.BigEndian.Uint16(data[i:i+2]) + 269
			i += 2
		} else if delta == 15 {
			return nil, errors.InvalidArgument("invalid option delta", nil)
		}
		if olen == 13 {
			if i >= len(data) {
				return nil, errors.InvalidArgument("truncated option length", nil)
			}
			olen = int(data[i]) + 13
			i++
		} else if olen == 14 {
			if i+1 >= len(data) {
				return nil, errors.InvalidArgument("truncated option length", nil)
			}
			olen = int(binary.BigEndian.Uint16(data[i:i+2])) + 269
			i += 2
		} else if olen == 15 {
			return nil, errors.InvalidArgument("invalid option length", nil)
		}
		if i+olen > len(data) {
			return nil, errors.InvalidArgument("truncated option value", nil)
		}
		num := lastOpt + delta
		lastOpt = num
		val := data[i : i+olen]
		i += olen
		switch num {
		case optUriPath:
			pathSegs = append(pathSegs, string(val))
		case optUriQuery:
			m.Query = string(val)
		}
	}
	if len(pathSegs) > 0 {
		m.Path = "/" + joinPath(pathSegs)
	}
	return m, nil
}

func splitPath(path string) []string {
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	if path == "" {
		return nil
	}
	var segs []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				segs = append(segs, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		segs = append(segs, path[start:])
	}
	return segs
}

func joinPath(segs []string) string {
	if len(segs) == 0 {
		return ""
	}
	out := segs[0]
	for i := 1; i < len(segs); i++ {
		out += "/" + segs[i]
	}
	return out
}
