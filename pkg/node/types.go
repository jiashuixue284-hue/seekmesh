package node

import (
	"context"
	"errors"
	"fmt"
)

type NodeID string

type PathType string

const (
	PathDirect PathType = "direct"
	PathRelay  PathType = "relay"
	PathMesh   PathType = "mesh"
)

var (
	ErrNoRouteToPeer = errors.New("no route to peer")
	ErrHopLimit      = errors.New("hop limit exceeded")
)

type Envelope struct {
	Source      NodeID
	Destination NodeID
	Payload     []byte
	PathType    PathType
	Route       []NodeID
	HopCount    int
}

type Session interface {
	PeerID() NodeID
	PathType() PathType
	Send(ctx context.Context, env Envelope) error
}

type Dialer interface {
	DialDirect(ctx context.Context, peer NodeID, candidates []string) (Session, error)
	DialRelay(ctx context.Context, relay NodeID, peer NodeID) (Session, error)
}

type DeliveryHandler func(env Envelope)

type RelayPicker interface {
	Relays() []NodeID
}

func (e Envelope) String() string {
	return fmt.Sprintf("src=%s dst=%s path=%s hops=%d route=%v", e.Source, e.Destination, e.PathType, e.HopCount, e.Route)
}
