package core

import (
	"crypto/ed25519"
	"math/rand"
	"sync"
)

// Message wrapper
type NetMsg struct {
	From NodeID
	To   NodeID
	Body interface{}
}

// Network deterministic tick-based scheduler
type Network struct {
	seed       int64
	rng        *rand.Rand
	minLatency int
	maxLatency int

	mu     sync.Mutex
	nodes  map[NodeID]*Node
	queues map[int][]NetMsg // tick -> messages
	curTick int
}

func NewNetwork(seed int64, minLat, maxLat int) *Network {
	r := rand.New(rand.NewSource(seed))
	return &Network{
		seed:      seed,
		rng:       r,
		minLatency: minLat,
		maxLatency: maxLat,
		queues:    make(map[int][]NetMsg),
		nodes:     make(map[NodeID]*Node),
	}
}

func (net *Network) SetNodes(nodes []*Node) {
	net.mu.Lock()
	defer net.mu.Unlock()
	for _, n := range nodes {
		net.nodes[n.id] = n
		// also set back-reference
		n.net = net
		if n.inbox == nil {
			n.inbox = make(chan NetMsg, 100)
		}
	}
}

// RegisterNode adds node to the network, sets node.net and inbox, and exchanges public keys
func RegisterNode(net *Network, n *Node) {
	net.mu.Lock()
	defer net.mu.Unlock()

	// attach network and inbox
	n.net = net
	if n.inbox == nil {
		n.inbox = make(chan NetMsg, 100)
	}

	// ensure node's pubs map exists
	if n.pubs == nil {
		n.pubs = make(map[NodeID]ed25519.PublicKey)
	}

	// exchange public keys with existing nodes
	for id, existing := range net.nodes {
		if existing == nil {
			continue
		}
		// ensure existing.pubs initialized
		if existing.pubs == nil {
			existing.pubs = make(map[NodeID]ed25519.PublicKey)
		}
		// if existing has a public key, add to new node
		if existing.kp.Pub != nil {
			n.pubs[id] = existing.kp.Pub
		}
		// if new node has pub, add to existing
		if n.kp.Pub != nil {
			existing.pubs[n.id] = n.kp.Pub
		}
	}

	// finally register node into network map
	if net.nodes == nil {
		net.nodes = make(map[NodeID]*Node)
	}
	net.nodes[n.id] = n

	// ensure node knows itself
	if n.kp.Pub != nil {
		n.pubs[n.id] = n.kp.Pub
	}
}

func (net *Network) Send(from, to NodeID, body interface{}) {
	// protect rng and queues with mutex
	net.mu.Lock()
	lat := net.minLatency
	if net.maxLatency > net.minLatency {
		lat += net.rng.Intn(net.maxLatency-net.minLatency+1)
	}
	deliveryTick := net.curTick + lat
	net.queues[deliveryTick] = append(net.queues[deliveryTick], NetMsg{From: from, To: to, Body: body})
	net.mu.Unlock()
}

func (net *Network) Broadcast(from NodeID, body interface{}) {
	net.mu.Lock()
	// copy keys to avoid holding lock during sends to channel (but we still need to append to queues)
	ids := make([]NodeID, 0, len(net.nodes))
	for id := range net.nodes {
		ids = append(ids, id)
	}
	net.mu.Unlock()

	// call Send for each recipient (Send itself locks for queues/rng)
	for _, id := range ids {
		net.Send(from, id, body)
	}
}

func (net *Network) Tick() {
	net.mu.Lock()
	net.curTick++
	msgs := net.queues[net.curTick]
	// Remove queue entry now under lock
	delete(net.queues, net.curTick)
	net.mu.Unlock()

	if len(msgs) == 0 {
		return
	}

	// deliver messages (non-blocking sends)
	for _, m := range msgs {
		net.mu.Lock()
		n, ok := net.nodes[m.To]
		net.mu.Unlock()
		if !ok || n == nil {
			continue
		}
		// non-blocking send
		select {
		case n.inbox <- m:
		default:
			// drop if full
		}
	}
}
