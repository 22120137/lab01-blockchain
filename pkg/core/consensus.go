package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"lab01/pkg/util"
)

// Node implements minimal proposer/voting

type Node struct {
	id     NodeID
	kp     Keypair
	pubs   map[NodeID]ed25519.PublicKey
	net    *Network
	inbox  chan NetMsg
	state  map[string]string
	height uint64

	pendingBlock *Block
	votes        map[uint64]map[string]map[VotePhase]map[NodeID]bool // height->blockHashHex->phase->voter
	Log          *util.Logger
	numNodes     int

	mu        sync.Mutex
	seed      int64
	finalized uint64
}

// NewNode creates a node, numNodes is the total validators count
func NewNode(id NodeID, seed int64, numNodes int, logger *util.Logger) *Node {
	kp := GenKeypair()
	n := &Node{
		id:     id,
		kp:     kp,
		pubs:   make(map[NodeID]ed25519.PublicKey),
		state:  make(map[string]string),
		votes:  make(map[uint64]map[string]map[VotePhase]map[NodeID]bool),
		Log:    logger,
		numNodes: numNodes,
		seed:   seed,
	}
	return n
}

func (n *Node) Start() {
	// set pub map to self for now; main will populate later if needed
	n.pubs[n.id] = n.kp.Pub
	go n.loop()
}

func (n *Node) loop() {
	for {
		select {
		case m := <-n.inbox:
			n.handleNetMsg(m)
		case <-time.After(200 * time.Millisecond):
			// idle - no op
		}
	}
}

func (n *Node) OnTick() {
	// simple proposer: if this node is proposer for next height, make block
	n.mu.Lock()
	h := n.height + 1
	proposer := NodeID(fmt.Sprintf("node%02d", int(h-1)%n.numNodes))
	if proposer == n.id {
		// make a block with a sample tx
		tx := Transaction{Sender: string(n.id), Key: string(n.id) + "/k", Value: fmt.Sprintf("v%d", h), Nonce: h}
		bt := MarshalTxCanonical(tx)
		tx.Sig = SignWithDomain("TX:", bt, n.kp.Priv)
		// apply locally
		blk := &Block{Header: BlockHeader{ParentHash: nil, Height: h, StateHash: nil, Proposer: n.id}, Txns: []Transaction{tx}}
		// compute state hash
		st := make(map[string]string)
		for k, v := range n.state {
			st[k] = v
		}
		_ = ApplyTx(st, tx)
		blk.Header.StateHash = StateHash(st)
		hb := MarshalHeaderCanonical(&blk.Header)
		blk.Header.Sig = SignWithDomain("HEADER:", hb, n.kp.Priv)
		// broadcast header first
		n.net.Broadcast(n.id, blk.Header)
		// schedule body a tick later (we use goroutine and sleep to simulate)
		go func(b *Block) {
			time.Sleep(10 * time.Millisecond)
			n.net.Broadcast(n.id, b)
		}(blk)
		n.Log.Printf("NODE|%s|PROPOSE|H=%d", n.id, h)
	}
	n.mu.Unlock()
}

func (n *Node) handleNetMsg(m NetMsg) {
	// handle types: BlockHeader, Block, Vote
	switch body := m.Body.(type) {
	case BlockHeader:
		n.Log.Printf("NODE|%s|RECV|HEADER|from=%s|H=%d", n.id, m.From, body.Height)
		// use proposer's pub key to verify
		pub, ok := n.pubs[body.Proposer]
		if !ok {
			n.Log.Printf("NODE|%s|RECV|HEADER|unknown_proposer=%s|H=%d", n.id, body.Proposer, body.Height)
			break
		}
		if VerifyWithDomain("HEADER:", MarshalHeaderCanonical(&body), body.Sig, pub) {
			// prevote
			hash := sha256.Sum256(MarshalHeaderCanonical(&body))
			v := Vote{Voter: n.id, Height: body.Height, BlockHash: hash[:], Phase: PhasePrevote}
			v.Sig = SignWithDomain("VOTE:", MarshalVoteCanonical(&v), n.kp.Priv)
			n.net.Broadcast(n.id, v)
			n.Log.Printf("NODE|%s|SENT|PREVOTE|H=%d", n.id, body.Height)
		} else {
			n.Log.Printf("NODE|%s|RECV|HEADER|badsig|proposer=%s|H=%d", n.id, body.Proposer, body.Height)
		}

		case Block:
		n.Log.Printf("NODE|%s|RECV|BLOCK|from=%s|H=%d|txs=%d", n.id, m.From, body.Header.Height, len(body.Txns))
		pub, ok := n.pubs[body.Header.Proposer]
		if !ok {
			n.Log.Printf("NODE|%s|RECV|BLOCK|unknown_proposer=%s|H=%d", n.id, body.Header.Proposer, body.Header.Height)
			break
		}
		if VerifyWithDomain("HEADER:", MarshalHeaderCanonical(&body.Header), body.Header.Sig, pub) {
			// for simplicity skip verifying tx sigs
			// prevote if not already
			hash := sha256.Sum256(MarshalHeaderCanonical(&body.Header))
			v := Vote{Voter: n.id, Height: body.Header.Height, BlockHash: hash[:], Phase: PhasePrevote}
			v.Sig = SignWithDomain("VOTE:", MarshalVoteCanonical(&v), n.kp.Priv)
			n.net.Broadcast(n.id, v)
			n.Log.Printf("NODE|%s|SENT|PREVOTE|H=%d|viaBLOCK", n.id, body.Header.Height)
		} else {
			n.Log.Printf("NODE|%s|RECV|BLOCK|badsig|proposer=%s|H=%d", n.id, body.Header.Proposer, body.Header.Height)
		}

	case Vote:
		n.Log.Printf("NODE|%s|RECV|VOTE|from=%s|H=%d|phase=%s", n.id, m.From, body.Height, body.Phase)
		// tally votes
		hashHex := string(body.BlockHash)
		n.mu.Lock()
		if _, ok := n.votes[body.Height]; !ok {
			n.votes[body.Height] = make(map[string]map[VotePhase]map[NodeID]bool)
		}
		if _, ok := n.votes[body.Height][hashHex]; !ok {
			n.votes[body.Height][hashHex] = make(map[VotePhase]map[NodeID]bool)
		}
		if _, ok := n.votes[body.Height][hashHex][body.Phase]; !ok {
			n.votes[body.Height][hashHex][body.Phase] = make(map[NodeID]bool)
		}
		n.votes[body.Height][hashHex][body.Phase][body.Voter] = true
		// check quorum
		num := len(n.votes[body.Height][hashHex][PhasePrevote])
		if num >= (n.numNodes/2 + 1) {
			// broadcast precommit if not already
			pc := Vote{Voter: n.id, Height: body.Height, BlockHash: []byte(hashHex), Phase: PhasePrecommit}
			pc.Sig = SignWithDomain("VOTE:", MarshalVoteCanonical(&pc), n.kp.Priv)
			n.net.Broadcast(n.id, pc)
			n.Log.Printf("NODE|%s|SENT|PRECOMMIT|H=%d", n.id, body.Height)
		}
		// count precommits
		numPC := len(n.votes[body.Height][hashHex][PhasePrecommit])
		if numPC >= (n.numNodes/2 + 1) {
			// finalize
			if body.Height > n.finalized {
				n.finalized = body.Height
				n.height = body.Height
				n.Log.Printf("NODE|%s|FINALIZED|H=%d", n.id, body.Height)
			}
		}
		n.mu.Unlock()
	default:
		// unknown
	}
}

func (n *Node) FinalizedHeight() uint64 { return n.finalized }
func (n *Node) SnapshotState() map[string]string {
	n.mu.Lock()
	defer n.mu.Unlock()
	copy := make(map[string]string)
	for k, v := range n.state {
		copy[k] = v
	}
	return copy
}
