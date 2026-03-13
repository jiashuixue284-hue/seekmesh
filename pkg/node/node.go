package node

import (
	"context"
	"sync"
)

type Config struct {
	ID          NodeID
	HopLimit    int
	RelayPicker RelayPicker
	Dialer      Dialer
	OnDeliver   DeliveryHandler
}

type Node struct {
	id          NodeID
	hopLimit    int
	relayPicker RelayPicker
	dialer      Dialer
	onDeliver   DeliveryHandler

	mu       sync.RWMutex
	sessions map[NodeID]Session
	topology *Topology
}

func New(cfg Config) *Node {
	hopLimit := cfg.HopLimit
	if hopLimit <= 0 {
		hopLimit = 8
	}
	n := &Node{
		id:          cfg.ID,
		hopLimit:    hopLimit,
		relayPicker: cfg.RelayPicker,
		dialer:      cfg.Dialer,
		onDeliver:   cfg.OnDeliver,
		sessions:    map[NodeID]Session{},
		topology:    NewTopology(),
	}
	n.topology.adj[n.id] = map[NodeID]linkInfo{}
	return n
}

func (n *Node) ID() NodeID { return n.id }

func (n *Node) Connect(ctx context.Context, peer NodeID, candidates []string) error {
	session, err := n.dialer.DialDirect(ctx, peer, candidates)
	if err == nil {
		n.bindSession(peer, session)
		n.topology.UpsertLink(n.id, peer, PathDirect)
		return nil
	}

	for _, relay := range n.relayPicker.Relays() {
		session, rerr := n.dialer.DialRelay(ctx, relay, peer)
		if rerr != nil {
			continue
		}
		n.bindSession(peer, session)
		n.topology.UpsertLink(n.id, peer, PathRelay)
		n.topology.UpsertLink(n.id, relay, PathDirect)
		return nil
	}
	return ErrNoRouteToPeer
}

func (n *Node) InjectTopologyLink(a, b NodeID, kind PathType) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.topology.UpsertLink(a, b, kind)
}

func (n *Node) Send(ctx context.Context, dst NodeID, payload []byte) error {
	env := Envelope{Source: n.id, Destination: dst, Payload: payload}
	return n.sendEnvelope(ctx, env)
}

func (n *Node) sendEnvelope(ctx context.Context, env Envelope) error {
	n.mu.RLock()
	s, ok := n.sessions[env.Destination]
	n.mu.RUnlock()
	if ok {
		env.PathType = s.PathType()
		env.Route = []NodeID{n.id, env.Destination}
		return s.Send(ctx, env)
	}

	n.mu.RLock()
	next, route, ok := n.topology.NextHop(n.id, env.Destination)
	if ok {
		s = n.sessions[next]
	}
	n.mu.RUnlock()
	if !ok || s == nil {
		return ErrNoRouteToPeer
	}
	env.PathType = PathMesh
	env.Route = route
	return s.Send(ctx, env)
}

func (n *Node) HandleIncoming(ctx context.Context, env Envelope) error {
	if env.Destination == n.id {
		if n.onDeliver != nil {
			n.onDeliver(env)
		}
		return nil
	}
	env.HopCount++
	if env.HopCount > n.hopLimit {
		return ErrHopLimit
	}
	return n.sendEnvelope(ctx, env)
}

func (n *Node) bindSession(peer NodeID, s Session) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sessions[peer] = s
}
