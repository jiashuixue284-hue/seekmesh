package node

import (
	"context"
	"errors"
	"testing"
)

type staticRelays []NodeID

func (s staticRelays) Relays() []NodeID { return []NodeID(s) }

type mockSession struct {
	peer NodeID
	path PathType
	sent []Envelope
}

func (m *mockSession) PeerID() NodeID                      { return m.peer }
func (m *mockSession) PathType() PathType                  { return m.path }
func (m *mockSession) Send(_ context.Context, e Envelope) error { m.sent = append(m.sent, e); return nil }

type mockDialer struct {
	direct map[NodeID]*mockSession
	relay  map[NodeID]*mockSession
}

func (m *mockDialer) DialDirect(_ context.Context, peer NodeID, _ []string) (Session, error) {
	if s, ok := m.direct[peer]; ok {
		return s, nil
	}
	return nil, errors.New("direct fail")
}

func (m *mockDialer) DialRelay(_ context.Context, _ NodeID, peer NodeID) (Session, error) {
	if s, ok := m.relay[peer]; ok {
		return s, nil
	}
	return nil, errors.New("relay fail")
}

func TestConnectPrefersDirect(t *testing.T) {
	d := &mockDialer{direct: map[NodeID]*mockSession{"peer": {peer: "peer", path: PathDirect}}, relay: map[NodeID]*mockSession{"peer": {peer: "peer", path: PathRelay}}}
	n := New(Config{ID: "n1", Dialer: d, RelayPicker: staticRelays{"r1"}})
	if err := n.Connect(context.Background(), "peer", nil); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := n.Send(context.Background(), "peer", []byte("hello")); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got := len(d.direct["peer"].sent); got != 1 {
		t.Fatalf("direct not used, sent=%d", got)
	}
	if got := len(d.relay["peer"].sent); got != 0 {
		t.Fatalf("relay should not be used, sent=%d", got)
	}
}

func TestConnectFallsBackToRelay(t *testing.T) {
	d := &mockDialer{direct: map[NodeID]*mockSession{}, relay: map[NodeID]*mockSession{"peer": {peer: "peer", path: PathRelay}}}
	n := New(Config{ID: "n1", Dialer: d, RelayPicker: staticRelays{"r1"}})
	if err := n.Connect(context.Background(), "peer", nil); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := n.Send(context.Background(), "peer", []byte("hello")); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got := len(d.relay["peer"].sent); got != 1 {
		t.Fatalf("relay not used, sent=%d", got)
	}
}

func TestMeshRoute(t *testing.T) {
	toB := &mockSession{peer: "b", path: PathDirect}
	d := &mockDialer{direct: map[NodeID]*mockSession{"b": toB}, relay: map[NodeID]*mockSession{}}
	n := New(Config{ID: "a", Dialer: d, RelayPicker: staticRelays{"r1"}})
	if err := n.Connect(context.Background(), "b", nil); err != nil {
		t.Fatalf("connect b: %v", err)
	}
	n.InjectTopologyLink("b", "c", PathDirect)
	if err := n.Send(context.Background(), "c", []byte("via-b")); err != nil {
		t.Fatalf("mesh send: %v", err)
	}
	if len(toB.sent) != 1 {
		t.Fatalf("expected forwarding via b")
	}
	if toB.sent[0].PathType != PathMesh {
		t.Fatalf("expected mesh path, got %s", toB.sent[0].PathType)
	}
	if toB.sent[0].Route[1] != "b" {
		t.Fatalf("unexpected route: %+v", toB.sent[0].Route)
	}
}
