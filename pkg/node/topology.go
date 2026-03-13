package node

import (
	"math"
)

type linkInfo struct {
	kind PathType
	cost int
}

type Topology struct {
	adj map[NodeID]map[NodeID]linkInfo
}

func NewTopology() *Topology {
	return &Topology{adj: map[NodeID]map[NodeID]linkInfo{}}
}

func (t *Topology) UpsertLink(a, b NodeID, kind PathType) {
	if a == "" || b == "" || a == b {
		return
	}
	cost := linkCost(kind)
	if t.adj[a] == nil {
		t.adj[a] = map[NodeID]linkInfo{}
	}
	if t.adj[b] == nil {
		t.adj[b] = map[NodeID]linkInfo{}
	}
	t.adj[a][b] = linkInfo{kind: kind, cost: cost}
	t.adj[b][a] = linkInfo{kind: kind, cost: cost}
}

func (t *Topology) NextHop(from, to NodeID) (NodeID, []NodeID, bool) {
	if from == to {
		return "", nil, false
	}
	dist := map[NodeID]int{}
	prev := map[NodeID]NodeID{}
	visited := map[NodeID]bool{}

	for n := range t.adj {
		dist[n] = math.MaxInt / 8
	}
	dist[from] = 0

	for {
		current := NodeID("")
		best := math.MaxInt / 8
		for n, d := range dist {
			if !visited[n] && d < best {
				best = d
				current = n
			}
		}
		if current == "" {
			break
		}
		if current == to {
			break
		}
		visited[current] = true
		for neighbor, info := range t.adj[current] {
			if visited[neighbor] {
				continue
			}
			alt := dist[current] + info.cost
			if alt < dist[neighbor] {
				dist[neighbor] = alt
				prev[neighbor] = current
			}
		}
	}

	if _, ok := dist[to]; !ok || dist[to] >= math.MaxInt/8 {
		return "", nil, false
	}

	path := []NodeID{to}
	for curr := to; curr != from; {
		p, ok := prev[curr]
		if !ok {
			return "", nil, false
		}
		path = append([]NodeID{p}, path...)
		curr = p
	}
	if len(path) < 2 {
		return "", nil, false
	}
	return path[1], path, true
}

func linkCost(kind PathType) int {
	switch kind {
	case PathDirect:
		return 1
	case PathRelay:
		return 3
	default:
		return 2
	}
}
