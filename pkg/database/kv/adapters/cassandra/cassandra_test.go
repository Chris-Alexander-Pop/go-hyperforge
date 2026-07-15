package cassandra_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv/adapters/cassandra"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/gocql/gocql"
)

// mockSession is an in-memory SessionAPI for unit tests.
type mockSession struct {
	mu    sync.Mutex
	store map[string][]byte
	ttls  map[string]time.Time
	fail  error
}

func newMockSession() *mockSession {
	return &mockSession{
		store: make(map[string][]byte),
		ttls:  make(map[string]time.Time),
	}
}

func (m *mockSession) QueryExec(_ context.Context, stmt string, args ...interface{}) error {
	if m.fail != nil {
		return m.fail
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// INSERT ... VALUES (?, ?) [USING TTL ?]
	if len(args) >= 2 && (contains(stmt, "INSERT") || contains(stmt, "insert")) {
		key, _ := args[0].(string)
		val, _ := args[1].([]byte)
		cp := append([]byte(nil), val...)
		m.store[key] = cp
		if len(args) >= 3 {
			if secs, ok := args[2].(int); ok && secs > 0 {
				m.ttls[key] = time.Now().Add(time.Duration(secs) * time.Second)
			}
		} else {
			delete(m.ttls, key)
		}
		return nil
	}
	// DELETE
	if contains(stmt, "DELETE") || contains(stmt, "delete") {
		if len(args) >= 1 {
			key, _ := args[0].(string)
			delete(m.store, key)
			delete(m.ttls, key)
		}
		return nil
	}
	return nil
}

func (m *mockSession) QueryScan(_ context.Context, stmt string, args []interface{}, dest ...interface{}) error {
	if m.fail != nil {
		return m.fail
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(args) < 1 {
		return gocql.ErrNotFound
	}
	key, _ := args[0].(string)
	val, ok := m.store[key]
	if !ok {
		return gocql.ErrNotFound
	}
	if exp, ok := m.ttls[key]; ok && time.Now().After(exp) {
		delete(m.store, key)
		delete(m.ttls, key)
		return gocql.ErrNotFound
	}

	if contains(stmt, "SELECT value") || contains(stmt, "select value") {
		if len(dest) > 0 {
			if p, ok := dest[0].(*[]byte); ok {
				*p = append([]byte(nil), val...)
			}
		}
		return nil
	}
	// SELECT key (Exists)
	if len(dest) > 0 {
		if p, ok := dest[0].(*string); ok {
			*p = key
		}
	}
	return nil
}

func (m *mockSession) Close() error { return nil }

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if equalFoldASCII(s[i:i+len(sub)], sub) {
					return true
				}
			}
			return false
		})())
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func TestNewFromSessionNil(t *testing.T) {
	_, err := cassandra.NewFromSession(nil, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestKVRoundTrip(t *testing.T) {
	a, err := cassandra.NewFromSession(newMockSession(), "ks", "kv")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	ctx := context.Background()
	if err := a.Set(ctx, "k1", []byte("v1"), 0); err != nil {
		t.Fatal(err)
	}
	got, err := a.Get(ctx, "k1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v1" {
		t.Fatalf("got %q", got)
	}
	ok, err := a.Exists(ctx, "k1")
	if err != nil || !ok {
		t.Fatalf("exists: %v %v", ok, err)
	}
	if err := a.Delete(ctx, "k1"); err != nil {
		t.Fatal(err)
	}
	ok, err = a.Exists(ctx, "k1")
	if err != nil || ok {
		t.Fatalf("exists after delete: %v %v", ok, err)
	}
	_, err = a.Get(ctx, "k1")
	if err == nil || !errors.IsCode(err, errors.CodeNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestSetWithTTL(t *testing.T) {
	a, err := cassandra.NewFromSession(newMockSession(), "ks", "kv")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ctx := context.Background()
	if err := a.Set(ctx, "ttl", []byte("x"), time.Hour); err != nil {
		t.Fatal(err)
	}
	got, err := a.Get(ctx, "ttl")
	if err != nil || string(got) != "x" {
		t.Fatalf("get: %v %q", err, got)
	}
}

func TestCanceledContext(t *testing.T) {
	a, err := cassandra.NewFromSession(newMockSession(), "ks", "kv")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Set(ctx, "k", []byte("v"), 0); err == nil {
		t.Fatal("expected canceled")
	}
}

func TestImplementsKV(t *testing.T) {
	a, _ := cassandra.NewFromSession(newMockSession(), "", "")
	var _ kv.KV = a
}

func TestIntegrationSkipShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cassandra integration in short mode")
	}
	// Live cluster optional; New will fail without one — skip unless host set.
	t.Skip("live Cassandra not provisioned in CI; unit mock covers SessionAPI path")
}
