package swim

import (
	"math/rand"
	"sync"
	"time"
)

// Educational sketch only — see package doc. Not production membership.

// State represents the state of a member.
type State int

const (
	Alive State = iota
	Suspect
	Dead
)

// Member represents a node in the cluster.
type Member struct {
	ID          string
	Address     string
	State       State
	Incarnation uint64
	LastUpdate  time.Time
}

// Config holds configuration for the Gossip protocol.
type Config struct {
	BindAddress    string
	ID             string
	ProtocolPeriod time.Duration
	PingTimeout    time.Duration
	SuspectTimeout time.Duration
	PingReqK       int // Number of members to ask to ping a suspect
}

// EventType classifies membership events.
type EventType string

const (
	EventJoin   EventType = "Join"
	EventLeave  EventType = "Leave"
	EventFail   EventType = "Fail"
	EventUpdate EventType = "Update"
)

// Protocol implements a basic SWIM-style gossip protocol logic.
// Transport is abstracted away; users must hook up networking.
type Protocol struct {
	config  Config
	members map[string]*Member
	mu      sync.RWMutex

	events chan Event
	stopCh chan struct{}
	done   chan struct{}

	// Local incarnation for refute.
	incarnation uint64

	// Transport hook
	Transport Transport
}

// Event is emitted on membership changes.
type Event struct {
	Type   EventType
	Member Member
}

// Transport abstracts network operations.
type Transport interface {
	Ping(target string) (bool, error)
	PingReq(target string, proxy string) (bool, error)
}

// New creates a new Gossip Protocol instance.
func New(config Config, transport Transport) *Protocol {
	if config.ProtocolPeriod == 0 {
		config.ProtocolPeriod = 1 * time.Second
	}
	if config.PingReqK == 0 {
		config.PingReqK = 3
	}
	if config.SuspectTimeout == 0 {
		config.SuspectTimeout = 3 * time.Second
	}

	return &Protocol{
		config:      config,
		members:     make(map[string]*Member),
		events:      make(chan Event, 100),
		stopCh:      make(chan struct{}),
		done:        make(chan struct{}),
		Transport:   transport,
		incarnation: 0,
	}
}

// Start starts the gossip loop.
func (p *Protocol) Start() {
	go p.loop()
}

// Stop stops the gossip loop and closes the Events channel after the loop exits.
func (p *Protocol) Stop() {
	select {
	case <-p.stopCh:
		return
	default:
		close(p.stopCh)
	}
	<-p.done
}

// Join adds a member to the local list (seeds) and emits a Join event.
func (p *Protocol) Join(id, address string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.members[id]; exists {
		return
	}
	m := &Member{
		ID:          id,
		Address:     address,
		State:       Alive,
		Incarnation: 0,
		LastUpdate:  time.Now(),
	}
	p.members[id] = m
	p.emitLocked(Event{Type: EventJoin, Member: *m})
}

// Members returns the list of known members.
func (p *Protocol) Members() []Member {
	p.mu.RLock()
	defer p.mu.RUnlock()

	list := make([]Member, 0, len(p.members))
	for _, m := range p.members {
		list = append(list, *m)
	}
	return list
}

// Incarnation returns this node's current incarnation number.
func (p *Protocol) Incarnation() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.incarnation
}

// GossipUpdate applies a remote membership update. If the update concerns this
// node and claims Suspect/Dead with an incarnation <= local, the node refutes
// by bumping its incarnation and emitting an Alive Update event.
func (p *Protocol) GossipUpdate(m Member) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if m.ID == p.config.ID {
		if m.State != Alive && m.Incarnation <= p.incarnation {
			p.incarnation++
			self := Member{
				ID:          p.config.ID,
				Address:     p.config.BindAddress,
				State:       Alive,
				Incarnation: p.incarnation,
				LastUpdate:  time.Now(),
			}
			p.emitLocked(Event{Type: EventUpdate, Member: self})
		}
		return
	}

	cur, exists := p.members[m.ID]
	if !exists {
		if m.State == Dead {
			return
		}
		cp := m
		cp.LastUpdate = time.Now()
		p.members[m.ID] = &cp
		p.emitLocked(Event{Type: EventJoin, Member: cp})
		return
	}

	// Prefer higher incarnation; ignore stale.
	if m.Incarnation < cur.Incarnation {
		return
	}
	if m.Incarnation == cur.Incarnation && stateRank(m.State) <= stateRank(cur.State) {
		return
	}

	cur.State = m.State
	cur.Incarnation = m.Incarnation
	cur.Address = m.Address
	cur.LastUpdate = time.Now()

	switch m.State {
	case Dead:
		delete(p.members, m.ID)
		p.emitLocked(Event{Type: EventFail, Member: *cur})
	case Suspect:
		p.emitLocked(Event{Type: EventUpdate, Member: *cur})
	default:
		p.emitLocked(Event{Type: EventUpdate, Member: *cur})
	}
}

func stateRank(s State) int {
	switch s {
	case Alive:
		return 0
	case Suspect:
		return 1
	case Dead:
		return 2
	default:
		return -1
	}
}

func (p *Protocol) emitLocked(ev Event) {
	select {
	case p.events <- ev:
	default:
		// Drop if buffer full (educational sketch).
	}
}

func (p *Protocol) loop() {
	defer close(p.done)
	defer close(p.events)

	ticker := time.NewTicker(p.config.ProtocolPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.probe()
		}
	}
}

// probe performs a gossip round (random ping).
func (p *Protocol) probe() {
	target := p.selectRandomMember()
	if target == nil {
		return
	}

	success, _ := p.Transport.Ping(target.Address)
	if !success {
		if !p.pingReq(target) {
			p.markSuspect(target.ID)
		}
	} else {
		p.markAlive(target.ID)
	}
}

func (p *Protocol) selectRandomMember() *Member {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.members) == 0 {
		return nil
	}

	keys := make([]string, 0, len(p.members))
	for k := range p.members {
		if k != p.config.ID {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}

	id := keys[rand.Intn(len(keys))]
	m := p.members[id]
	cp := *m
	return &cp
}

func (p *Protocol) pingReq(target *Member) bool {
	p.mu.RLock()
	proxies := make([]*Member, 0)
	for _, m := range p.members {
		if m.ID != p.config.ID && m.ID != target.ID && m.State == Alive {
			proxies = append(proxies, m)
		}
	}
	p.mu.RUnlock()

	rand.Shuffle(len(proxies), func(i, j int) { proxies[i], proxies[j] = proxies[j], proxies[i] })
	k := p.config.PingReqK
	if len(proxies) < k {
		k = len(proxies)
	}

	for i := 0; i < k; i++ {
		proxy := proxies[i]
		ok, _ := p.Transport.PingReq(target.Address, proxy.Address)
		if ok {
			return true
		}
	}
	return false
}

func (p *Protocol) markSuspect(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	m, ok := p.members[id]
	if !ok {
		return
	}

	if m.State == Alive {
		m.State = Suspect
		m.LastUpdate = time.Now()
		p.emitLocked(Event{Type: EventUpdate, Member: *m})
	} else if m.State == Suspect {
		if time.Since(m.LastUpdate) > p.config.SuspectTimeout {
			m.State = Dead
			cp := *m
			delete(p.members, m.ID)
			p.emitLocked(Event{Type: EventFail, Member: cp})
		}
	}
}

func (p *Protocol) markAlive(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	m, ok := p.members[id]
	if !ok {
		return
	}
	if m.State != Alive {
		m.State = Alive
		m.Incarnation++
		m.LastUpdate = time.Now()
		p.emitLocked(Event{Type: EventUpdate, Member: *m})
	}
}

// Events returns the event channel.
func (p *Protocol) Events() <-chan Event {
	return p.events
}
