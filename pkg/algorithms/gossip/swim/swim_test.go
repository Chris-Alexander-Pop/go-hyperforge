package swim_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/gossip/swim"
)

type okTransport struct{}

func (okTransport) Ping(string) (bool, error)             { return true, nil }
func (okTransport) PingReq(string, string) (bool, error) { return true, nil }

type failTransport struct{}

func (failTransport) Ping(string) (bool, error)             { return false, nil }
func (failTransport) PingReq(string, string) (bool, error) { return false, nil }

func TestJoinEmitsEventAndStop(t *testing.T) {
	p := swim.New(swim.Config{
		ID:             "self",
		BindAddress:    "127.0.0.1:1",
		ProtocolPeriod: 50 * time.Millisecond,
	}, okTransport{})
	p.Start()
	defer p.Stop()

	p.Join("peer", "127.0.0.1:2")

	select {
	case ev := <-p.Events():
		if ev.Type != swim.EventJoin || ev.Member.ID != "peer" {
			t.Fatalf("event=%+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Join event")
	}
}

func TestIncarnationRefute(t *testing.T) {
	p := swim.New(swim.Config{
		ID:             "self",
		BindAddress:    "127.0.0.1:1",
		ProtocolPeriod: time.Hour,
	}, okTransport{})
	p.Start()
	defer p.Stop()

	p.GossipUpdate(swim.Member{
		ID:          "self",
		Address:     "127.0.0.1:1",
		State:       swim.Suspect,
		Incarnation: 0,
	})

	select {
	case ev := <-p.Events():
		if ev.Type != swim.EventUpdate || ev.Member.State != swim.Alive {
			t.Fatalf("event=%+v", ev)
		}
		if ev.Member.Incarnation != 1 {
			t.Fatalf("incarnation=%d want 1", ev.Member.Incarnation)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for refute")
	}

	if p.Incarnation() != 1 {
		t.Fatalf("Incarnation=%d", p.Incarnation())
	}
}

func TestSuspectEmitsUpdate(t *testing.T) {
	p := swim.New(swim.Config{
		ID:             "self",
		BindAddress:    "127.0.0.1:1",
		ProtocolPeriod: 20 * time.Millisecond,
		SuspectTimeout: time.Hour,
		PingReqK:       1,
	}, failTransport{})
	p.Start()
	defer p.Stop()

	// Drain join event.
	p.Join("peer", "127.0.0.1:2")
	select {
	case <-p.Events():
	case <-time.After(time.Second):
		t.Fatal("no join")
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-p.Events():
			if ev.Type == swim.EventUpdate && ev.Member.ID == "peer" && ev.Member.State == swim.Suspect {
				return
			}
		case <-deadline:
			t.Fatal("timeout waiting for Suspect update")
		}
	}
}
