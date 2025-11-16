package core

import (
	"crypto/ed25519"
	"fmt"
	"math/rand"
	"sort"
	"sync"

	"lab01/pkg/util"
)

// Message wrapper
type NetMsg struct {
	From NodeID
	To   NodeID
	Body interface{}
}

// Network deterministic tick-based scheduler
type Network struct {
	seed               int64
	rng                *rand.Rand
	minLatency         int
	maxLatency         int
	dropRate           float64
	duplicateRate      float64
	maxQueuePerTick    int
	logger             *util.Logger
	maxOutboundPerTick int
	blockDurationTicks int

	mu           sync.Mutex
	nodes        map[NodeID]*Node
	queues       map[int][]NetMsg // tick -> messages
	curTick      int
	sentThisTick map[NodeID]int
	blockedPeers map[NodeID]int
}

func NewNetwork(seed int64, minLat, maxLat int, dropRate, duplicateRate float64, maxQueue int, logger *util.Logger, maxOutbound, blockDuration int) *Network {
	r := rand.New(rand.NewSource(seed))
	if maxLat < minLat {
		maxLat = minLat
	}
	if maxQueue <= 0 {
		maxQueue = 1024
	}
	if maxOutbound < 0 {
		maxOutbound = 0
	}
	if blockDuration <= 0 {
		blockDuration = 5
	}
	return &Network{
		seed:               seed,
		rng:                r,
		minLatency:         minLat,
		maxLatency:         maxLat,
		dropRate:           dropRate,
		duplicateRate:      duplicateRate,
		maxQueuePerTick:    maxQueue,
		logger:             logger,
		maxOutboundPerTick: maxOutbound,
		blockDurationTicks: blockDuration,
		queues:             make(map[int][]NetMsg),
		nodes:              make(map[NodeID]*Node),
		sentThisTick:       make(map[NodeID]int),
		blockedPeers:       make(map[NodeID]int),
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

func (net *Network) logf(format string, a ...interface{}) {
	if net.logger == nil {
		return
	}
	net.logger.Printf("NET|"+format, a...)
}

func (net *Network) logEvent(event string, from, to NodeID, body interface{}, extra string) {
	if net.logger == nil {
		return
	}
	height, _ := extractHeight(body)
	msg := fmt.Sprintf("event=%s|from=%s|to=%s|tick=%d|height=%d", event, from, to, net.curTick, height)
	if extra != "" {
		msg += "|" + extra
	}
	net.logger.Printf("NET|%s", msg)
}

func (net *Network) enqueue(tick int, msg NetMsg) bool {
	if net.maxQueuePerTick > 0 && len(net.queues[tick]) >= net.maxQueuePerTick {
		return false
	}
	net.queues[tick] = append(net.queues[tick], msg)
	return true
}

func (net *Network) Send(from, to NodeID, body interface{}) {
	net.mu.Lock()
	defer net.mu.Unlock()
	if until, ok := net.blockedPeers[from]; ok && net.curTick >= until {
		delete(net.blockedPeers, from)
		net.logEvent("UNBLOCK", from, "", nil, fmt.Sprintf("tick=%d", net.curTick))
	}
	if until, ok := net.blockedPeers[from]; ok {
		net.logEvent("BLOCKED", from, to, body, fmt.Sprintf("until=%d", until))
		return
	}

	if net.maxOutboundPerTick > 0 {
		net.sentThisTick[from]++
		if net.sentThisTick[from] > net.maxOutboundPerTick {
			net.blockedPeers[from] = net.curTick + net.blockDurationTicks
			net.logEvent("RATE_BLOCK", from, to, body, fmt.Sprintf("until=%d", net.blockedPeers[from]))
			return
		}
	}

	lat := net.minLatency
	if net.maxLatency > net.minLatency {
		lat += net.rng.Intn(net.maxLatency - net.minLatency + 1)
	}
	deliveryTick := net.curTick + lat
	if net.dropRate > 0 && net.rng.Float64() < net.dropRate {
		net.logEvent("DROP", from, to, body, fmt.Sprintf("lat=%d", lat))
		return
	}
	msg := NetMsg{From: from, To: to, Body: body}
	if !net.enqueue(deliveryTick, msg) {
		net.logEvent("QUEUE_FULL", from, to, body, fmt.Sprintf("deliver_tick=%d", deliveryTick))
		return
	}
	net.logEvent("SEND", from, to, body, fmt.Sprintf("deliver_tick=%d|type=%T", deliveryTick, body))
	if net.duplicateRate > 0 && net.rng.Float64() < net.duplicateRate {
		if net.enqueue(deliveryTick, msg) {
			net.logEvent("DUP", from, to, body, fmt.Sprintf("tick=%d", deliveryTick))
		} else {
			net.logEvent("DUP_DROP", from, to, body, fmt.Sprintf("tick=%d", deliveryTick))
		}
	}
}

func (net *Network) sendWithExtraDelay(from, to NodeID, body interface{}, extra int) {
	net.mu.Lock()
	defer net.mu.Unlock()
	if until, ok := net.blockedPeers[from]; ok && net.curTick >= until {
		delete(net.blockedPeers, from)
		net.logEvent("UNBLOCK", from, "", nil, fmt.Sprintf("tick=%d", net.curTick))
	}
	if until, ok := net.blockedPeers[from]; ok {
		net.logEvent("BLOCKED", from, to, body, fmt.Sprintf("until=%d", until))
		return
	}
	if net.maxOutboundPerTick > 0 {
		net.sentThisTick[from]++
		if net.sentThisTick[from] > net.maxOutboundPerTick {
			net.blockedPeers[from] = net.curTick + net.blockDurationTicks
			net.logEvent("RATE_BLOCK", from, to, body, fmt.Sprintf("until=%d", net.blockedPeers[from]))
			return
		}
	}

	lat := net.minLatency
	if net.maxLatency > net.minLatency {
		lat += net.rng.Intn(net.maxLatency - net.minLatency + 1)
	}
	if extra > 0 {
		lat += extra
	}
	deliveryTick := net.curTick + lat
	if net.dropRate > 0 && net.rng.Float64() < net.dropRate {
		net.logEvent("DROP", from, to, body, fmt.Sprintf("lat=%d", lat))
		return
	}
	msg := NetMsg{From: from, To: to, Body: body}
	if !net.enqueue(deliveryTick, msg) {
		net.logEvent("QUEUE_FULL", from, to, body, fmt.Sprintf("deliver_tick=%d", deliveryTick))
		return
	}
	net.logEvent("SEND", from, to, body, fmt.Sprintf("deliver_tick=%d|type=%T", deliveryTick, body))
	if net.duplicateRate > 0 && net.rng.Float64() < net.duplicateRate {
		if net.enqueue(deliveryTick, msg) {
			net.logEvent("DUP", from, to, body, fmt.Sprintf("tick=%d", deliveryTick))
		} else {
			net.logEvent("DUP_DROP", from, to, body, fmt.Sprintf("tick=%d", deliveryTick))
		}
	}
}

func (net *Network) Broadcast(from NodeID, body interface{}) {
	net.mu.Lock()
	// copy keys to avoid holding lock during sends to channel (but we still need to append to queues)
	ids := make([]NodeID, 0, len(net.nodes))
	for id := range net.nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return string(ids[i]) < string(ids[j]) })
	net.mu.Unlock()

	// call Send for each recipient (Send itself locks for queues/rng)
	for _, id := range ids {
		net.Send(from, id, body)
	}
}

func (net *Network) BroadcastWithDelay(from NodeID, body interface{}, extra int) {
	net.mu.Lock()
	ids := make([]NodeID, 0, len(net.nodes))
	for id := range net.nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return string(ids[i]) < string(ids[j]) })
	net.mu.Unlock()

	for _, id := range ids {
		net.sendWithExtraDelay(from, id, body, extra)
	}
}

func (net *Network) Tick() {
	net.mu.Lock()
	net.curTick++
	net.sentThisTick = make(map[NodeID]int)
	if len(net.blockedPeers) > 0 {
		ids := make([]NodeID, 0, len(net.blockedPeers))
		for id := range net.blockedPeers {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return string(ids[i]) < string(ids[j]) })
		for _, id := range ids {
			if net.curTick >= net.blockedPeers[id] {
				delete(net.blockedPeers, id)
				net.logEvent("UNBLOCK", id, "", nil, fmt.Sprintf("tick=%d", net.curTick))
			}
		}
	}
	msgs := net.queues[net.curTick]
	// Remove queue entry now under lock
	delete(net.queues, net.curTick)
	net.mu.Unlock()

	if len(msgs) == 0 {
		return
	}
	sort.SliceStable(msgs, func(i, j int) bool {
		if msgs[i].From != msgs[j].From {
			return string(msgs[i].From) < string(msgs[j].From)
		}
		if msgs[i].To != msgs[j].To {
			return string(msgs[i].To) < string(msgs[j].To)
		}
		ti := fmt.Sprintf("%T", msgs[i].Body)
		tj := fmt.Sprintf("%T", msgs[j].Body)
		if ti != tj {
			return ti < tj
		}
		return i < j
	})

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
			net.logEvent("DELIVER", m.From, m.To, m.Body, fmt.Sprintf("type=%T", m.Body))
		default:
			// drop if full
			net.logEvent("INBOX_FULL", m.From, m.To, m.Body, "")
		}
	}
}

func extractHeight(body interface{}) (uint64, bool) {
	switch b := body.(type) {
	case BlockHeader:
		return b.Height, true
	case *BlockHeader:
		if b == nil {
			return 0, false
		}
		return b.Height, true
	case Block:
		return b.Header.Height, true
	case *Block:
		if b == nil {
			return 0, false
		}
		return b.Header.Height, true
	case Vote:
		return b.Height, true
	case *Vote:
		if b == nil {
			return 0, false
		}
		return b.Height, true
	default:
		return 0, false
	}
}
