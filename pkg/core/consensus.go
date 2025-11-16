package core

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
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
	state  *State
	height uint64

	votes             map[uint64]map[string]map[VotePhase]map[NodeID]bool // height->blockHashHex->phase->voter
	pendingBlocks     map[uint64]map[string]*Block
	ledger            []Block
	lastBlockHash     []byte
	sentPrecommit     map[uint64]string
	txPool            map[string]Transaction // key sender/nonce
	poolNonce         map[string]uint64
	maxTxPerBlock     int
	proposalInterval  uint64
	lastProposalTick  map[uint64]uint64
	currentTick       uint64
	nextSelfTxTick    uint64
	rounds            map[uint64]uint64
	lastVoteTick      map[uint64]uint64
	Log               *util.Logger
	numNodes          int

	mu        sync.Mutex
	seed      int64
	finalized uint64
}

// NewNode creates a node, numNodes is the total validators count
func NewNode(id NodeID, seed int64, numNodes int, logger *util.Logger) *Node {
	kp := GenKeypair()
	n := &Node{
		id:               id,
		kp:               kp,
		pubs:             make(map[NodeID]ed25519.PublicKey),
		state:            NewState(),
		votes:            make(map[uint64]map[string]map[VotePhase]map[NodeID]bool),
		pendingBlocks:    make(map[uint64]map[string]*Block),
		ledger:           make([]Block, 0),
		lastBlockHash:    nil,
		sentPrecommit:    make(map[uint64]string),
		txPool:           make(map[string]Transaction),
		poolNonce:        make(map[string]uint64),
		maxTxPerBlock:    8,
		proposalInterval: 5,
		lastProposalTick: make(map[uint64]uint64),
		currentTick:      0,
		nextSelfTxTick:   0,
		rounds:           make(map[uint64]uint64),
		lastVoteTick:     make(map[uint64]uint64),
		Log:              logger,
		numNodes:         numNodes,
		seed:             seed,
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
	n.mu.Lock()
	n.currentTick++
	tick := n.currentTick
	n.mu.Unlock()

	n.maybeGenerateSelfTx(tick)

	n.mu.Lock()
	h := n.height + 1
	proposer := NodeID(fmt.Sprintf("node%02d", int(h-1)%n.numNodes))
	if proposer == n.id {
		if last := n.lastProposalTick[h]; last != 0 && tick-last < n.proposalInterval {
			n.mu.Unlock()
			return
		}
		n.lastProposalTick[h] = tick
		parentHash := cloneBytes(n.lastBlockHash)
		blockTxs := n.selectTxsForBlockLocked()
		st := n.state.Clone()
		for _, tx := range blockTxs {
			_ = ApplyTx(st, tx)
		}
		blk := &Block{Header: BlockHeader{ParentHash: parentHash, Height: h, StateHash: nil, Proposer: n.id}, Txns: blockTxs}
		blk.Header.StateHash = StateHash(st)
		hb := MarshalHeaderCanonical(&blk.Header)
		blk.Header.Sig = SignWithDomain("HEADER:", hb, n.kp.Priv)
		hashHex := hex.EncodeToString(HashBlockHeader(&blk.Header))
		n.storePendingBlockLocked(h, hashHex, blk)
		n.mu.Unlock()
		n.net.Broadcast(n.id, blk.Header)
		go func(b *Block) {
			time.Sleep(10 * time.Millisecond)
			n.net.Broadcast(n.id, b)
		}(blk)
		n.Log.Printf("NODE|%s|PROPOSE|H=%d|txs=%d", n.id, h, len(blockTxs))
		return
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
			// ensure parent matches current finalized view
			n.mu.Lock()
			validParent := n.isValidParentLocked(body.ParentHash, body.Height)
			n.mu.Unlock()
			if !validParent {
				n.Log.Printf("NODE|%s|RECV|HEADER|bad_parent|H=%d", n.id, body.Height)
				break
			}
			// prevote
			hash := sha256.Sum256(MarshalHeaderCanonical(&body))
			n.broadcastPrevote(body.Height, hash[:])
		} else {
			n.Log.Printf("NODE|%s|RECV|HEADER|badsig|proposer=%s|H=%d", n.id, body.Proposer, body.Height)
		}

	case Block:
		b := body
		n.handleBlockMsg(&b, m.From)
	case *Block:
		n.handleBlockMsg(body, m.From)
	case Transaction:
		tx := body
		n.handleTxMessage(&tx, m.From)
	case *Transaction:
		n.handleTxMessage(body, m.From)

	case Vote:
		n.Log.Printf("NODE|%s|RECV|VOTE|from=%s|H=%d|phase=%s", n.id, m.From, body.Height, body.Phase)
		pub, ok := n.pubs[body.Voter]
		if !ok {
			n.Log.Printf("NODE|%s|RECV|VOTE|unknown_voter=%s|H=%d", n.id, body.Voter, body.Height)
			break
		}
		if !VerifyWithDomain("VOTE:", MarshalVoteCanonical(&body), body.Sig, pub) {
			n.Log.Printf("NODE|%s|RECV|VOTE|badsig|from=%s|H=%d", n.id, body.Voter, body.Height)
			break
		}
		// tally votes
		hashHex := hex.EncodeToString(body.BlockHash)
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
			if n.sentPrecommit[body.Height] != hashHex {
				pc := Vote{Voter: n.id, Height: body.Height, BlockHash: body.BlockHash, Phase: PhasePrecommit}
				pc.Sig = SignWithDomain("VOTE:", MarshalVoteCanonical(&pc), n.kp.Priv)
				n.net.Broadcast(n.id, pc)
				n.sentPrecommit[body.Height] = hashHex
				n.Log.Printf("NODE|%s|SENT|PRECOMMIT|H=%d", n.id, body.Height)
			}
		}
		n.tryFinalizeLocked(body.Height, hashHex)
		n.mu.Unlock()
	default:
		// unknown
	}
}

func (n *Node) handleBlockMsg(body *Block, from NodeID) {
	n.Log.Printf("NODE|%s|RECV|BLOCK|from=%s|H=%d|txs=%d", n.id, from, body.Header.Height, len(body.Txns))
	pub, ok := n.pubs[body.Header.Proposer]
	if !ok {
		n.Log.Printf("NODE|%s|RECV|BLOCK|unknown_proposer=%s|H=%d", n.id, body.Header.Proposer, body.Header.Height)
		return
	}
	if !VerifyWithDomain("HEADER:", MarshalHeaderCanonical(&body.Header), body.Header.Sig, pub) {
		n.Log.Printf("NODE|%s|RECV|BLOCK|badsig|proposer=%s|H=%d", n.id, body.Header.Proposer, body.Header.Height)
		return
	}
	n.mu.Lock()
	validParent := n.isValidParentLocked(body.Header.ParentHash, body.Header.Height)
	n.mu.Unlock()
	if !validParent {
		n.Log.Printf("NODE|%s|RECV|BLOCK|bad_parent|H=%d", n.id, body.Header.Height)
		return
	}
	if !n.verifyBlockTransactions(*body) {
		return
	}
	hash := HashBlockHeader(&body.Header)
	hashHex := hex.EncodeToString(hash)
	n.mu.Lock()
	n.storePendingBlockLocked(body.Header.Height, hashHex, body)
	n.mu.Unlock()
	n.broadcastPrevote(body.Header.Height, hash)
}

func (n *Node) handleTxMessage(tx *Transaction, from NodeID) {
	if tx == nil {
		return
	}
	if n.addTxToPool(*tx) {
		n.Log.Printf("NODE|%s|RECV|TX|from=%s|sender=%s|nonce=%d", n.id, from, tx.Sender, tx.Nonce)
	}
}

func (n *Node) maybeGenerateSelfTx(tick uint64) {
	n.mu.Lock()
	if tick < n.nextSelfTxTick {
		n.mu.Unlock()
		return
	}
	sender := string(n.id)
	base := n.state.Nonces[sender]
	if v := n.poolNonce[sender]; v > base {
		base = v
	}
	nonce := base + 1
	key := fmt.Sprintf("%s/auto%d", sender, nonce)
	value := fmt.Sprintf("val%d", nonce)
	n.nextSelfTxTick = tick + 5
	n.mu.Unlock()

	tx := Transaction{Sender: sender, Key: key, Value: value, Nonce: nonce}
	bt := MarshalTxCanonical(tx)
	tx.Sig = SignWithDomain("TX:", bt, n.kp.Priv)
	if n.addTxToPool(tx) {
		n.net.Broadcast(n.id, tx)
		n.Log.Printf("NODE|%s|ENQUEUE_TX|nonce=%d", n.id, nonce)
	}
}

func (n *Node) selectTxsForBlockLocked() []Transaction {
	if len(n.txPool) == 0 {
		return nil
	}
	keys := make([]string, 0, len(n.txPool))
	for k := range n.txPool {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	st := n.state.Clone()
	selected := make([]Transaction, 0, n.maxTxPerBlock)
	for _, key := range keys {
		if len(selected) >= n.maxTxPerBlock {
			break
		}
		tx := n.txPool[key]
		if err := ApplyTx(st, tx); err != nil {
			continue
		}
		selected = append(selected, tx)
	}
	return selected
}

func (n *Node) addTxToPool(tx Transaction) bool {
	if !ownsKey(tx) {
		return false
	}
	pub, ok := n.pubs[NodeID(tx.Sender)]
	if !ok {
		return false
	}
	if !VerifyWithDomain("TX:", MarshalTxCanonical(tx), tx.Sig, pub) {
		return false
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	base := n.state.Nonces[tx.Sender]
	if v := n.poolNonce[tx.Sender]; v > base {
		base = v
	}
	if tx.Nonce != base+1 {
		return false
	}
	key := txPoolKey(tx.Sender, tx.Nonce)
	if _, exists := n.txPool[key]; exists {
		return false
	}
	n.txPool[key] = tx
	n.poolNonce[tx.Sender] = tx.Nonce
	return true
}

func (n *Node) removeTxsFromPoolLocked(txs []Transaction) {
	if len(txs) == 0 {
		return
	}
	affected := make(map[string]bool)
	for _, tx := range txs {
		key := txPoolKey(tx.Sender, tx.Nonce)
		delete(n.txPool, key)
		affected[tx.Sender] = true
	}
	for sender := range affected {
		maxNonce := n.state.Nonces[sender]
		for key, tx := range n.txPool {
			if strings.HasPrefix(key, sender+"/") {
				if tx.Nonce > maxNonce {
					maxNonce = tx.Nonce
				}
			}
		}
		if maxNonce == 0 {
			delete(n.poolNonce, sender)
		} else {
			n.poolNonce[sender] = maxNonce
		}
	}
}

func (n *Node) broadcastPrevote(height uint64, hash []byte) {
	v := Vote{Voter: n.id, Height: height, BlockHash: hash, Phase: PhasePrevote}
	v.Sig = SignWithDomain("VOTE:", MarshalVoteCanonical(&v), n.kp.Priv)
	n.net.Broadcast(n.id, v)
	n.Log.Printf("NODE|%s|SENT|PREVOTE|H=%d", n.id, height)
}

func (n *Node) storePendingBlockLocked(height uint64, hashHex string, blk *Block) {
	if _, ok := n.pendingBlocks[height]; !ok {
		n.pendingBlocks[height] = make(map[string]*Block)
	}
	headerCopy := BlockHeader{
		ParentHash: cloneBytes(blk.Header.ParentHash),
		Height:     blk.Header.Height,
		StateHash:  cloneBytes(blk.Header.StateHash),
		Proposer:   blk.Header.Proposer,
		Sig:        cloneBytes(blk.Header.Sig),
	}
	txCopy := make([]Transaction, len(blk.Txns))
	for i, tx := range blk.Txns {
		copyTx := tx
		copyTx.Sig = cloneBytes(tx.Sig)
		txCopy[i] = copyTx
	}
	copyBlock := &Block{Header: headerCopy, Txns: txCopy}
	n.pendingBlocks[height][hashHex] = copyBlock
	n.tryFinalizeLocked(height, hashHex)
}

func (n *Node) tryFinalizeLocked(height uint64, hashHex string) {
	if height <= n.finalized {
		return
	}
	blocks, ok := n.pendingBlocks[height]
	if !ok {
		return
	}
	blk := blocks[hashHex]
	if blk == nil {
		return
	}
	phaseByHash, ok := n.votes[height]
	if !ok {
		return
	}
	phaseMap, ok := phaseByHash[hashHex]
	if !ok {
		return
	}
	if len(phaseMap[PhasePrecommit]) < (n.numNodes/2 + 1) {
		return
	}

	n.applyBlock(blk)
	n.finalized = height
	n.height = height
	n.lastBlockHash = HashBlockHeader(&blk.Header)
	n.ledger = append(n.ledger, *blk)
	n.removeTxsFromPoolLocked(blk.Txns)
	delete(blocks, hashHex)
	if len(blocks) == 0 {
		delete(n.pendingBlocks, height)
	}
	n.Log.Printf("NODE|%s|FINALIZED|H=%d", n.id, height)
}

func (n *Node) verifyBlockTransactions(blk Block) bool {
	n.mu.Lock()
	temp := n.state.Clone()
	n.mu.Unlock()
	for _, tx := range blk.Txns {
		senderID := NodeID(tx.Sender)
		pub, ok := n.pubs[senderID]
		if !ok {
			n.Log.Printf("NODE|%s|BLOCK|unknown_sender=%s|H=%d", n.id, tx.Sender, blk.Header.Height)
			return false
		}
		if !VerifyWithDomain("TX:", MarshalTxCanonical(tx), tx.Sig, pub) {
			n.Log.Printf("NODE|%s|BLOCK|badsig_tx|sender=%s|H=%d", n.id, tx.Sender, blk.Header.Height)
			return false
		}
		if err := ApplyTx(temp, tx); err != nil {
			n.Log.Printf("NODE|%s|BLOCK|invalid_tx|err=%v", n.id, err)
			return false
		}
	}
	stateHash := StateHash(temp)
	if !bytes.Equal(stateHash, blk.Header.StateHash) {
		n.Log.Printf("NODE|%s|BLOCK|statehash_mismatch|H=%d", n.id, blk.Header.Height)
		return false
	}
	return true
}

func (n *Node) isValidParentLocked(parent []byte, height uint64) bool {
	if height == 1 {
		return len(parent) == 0
	}
	return bytes.Equal(parent, n.lastBlockHash)
}

func (n *Node) applyBlock(blk *Block) {
	for _, tx := range blk.Txns {
		if err := ApplyTx(n.state, tx); err != nil {
			n.Log.Printf("NODE|%s|APPLY|err=%v", n.id, err)
			return
		}
	}
}

func cloneBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}

func (n *Node) ID() NodeID {
	return n.id
}

func txPoolKey(sender string, nonce uint64) string {
	return fmt.Sprintf("%s/%d", sender, nonce)
}

func ownsKey(tx Transaction) bool {
	prefix := tx.Sender + "/"
	return strings.HasPrefix(tx.Key, prefix)
}

func (n *Node) FinalizedHeight() uint64 { return n.finalized }

type StateSnapshot struct {
	Data   map[string]string `json:"data"`
	Nonces map[string]uint64 `json:"nonces"`
}

func (n *Node) SnapshotState() StateSnapshot {
	n.mu.Lock()
	defer n.mu.Unlock()
	clone := n.state.Clone()
	return StateSnapshot{
		Data:   clone.Data,
		Nonces: clone.Nonces,
	}
}
