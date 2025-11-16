package core

import (
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"testing"

	"lab01/pkg/util"
)

const testChainID = "test-chain"

func TestConsensus_SimpleFinalization(t *testing.T) {
	logger := util.NewDeterministicLogger(io.Discard)
	net := NewNetwork(12345, 1, 1, 0, 0, 64, logger, 0, 5)
	nodes := buildTestNodes(net, logger, 4, 12345)

	runTicks(net, nodes, 120)

	for i, n := range nodes {
		if n.FinalizedHeight() < 1 {
			t.Fatalf("node %d did not finalize height 1", i)
		}
	}
}

func TestConsensusRejectsInvalidHeaderSignature(t *testing.T) {
	logger := util.NewDeterministicLogger(io.Discard)
	n := NewNode(NodeID("node00"), 1, 4, testChainID, logger)
	kp := GenKeypair()
	n.pubs[NodeID("node01")] = kp.Pub

	h := BlockHeader{Height: 1, Proposer: NodeID("node01")}
	h.Sig = SignWithDomain("HEADER:"+testChainID, MarshalHeaderCanonical(&h), n.kp.Priv)

	n.handleNetMsg(NetMsg{From: NodeID("node01"), Body: h})
	if len(n.votes) != 0 {
		t.Fatalf("invalid header signature should not produce votes")
	}
}

func TestConsensusIgnoresDuplicateVotes(t *testing.T) {
	logger := util.NewDeterministicLogger(io.Discard)
	n := NewNode(NodeID("node00"), 1, 4, testChainID, logger)
	kp := GenKeypair()
	n.pubs[NodeID("node01")] = kp.Pub

	v := Vote{Voter: NodeID("node01"), Height: 1, BlockHash: []byte("block"), Phase: PhasePrevote}
	v.Sig = SignWithDomain("VOTE:"+testChainID, MarshalVoteCanonical(&v), kp.Priv)

	msg := NetMsg{From: NodeID("node01"), Body: v}
	n.handleNetMsg(msg)
	n.handleNetMsg(msg)
	hashHex := hex.EncodeToString(v.BlockHash)
	if got := len(n.votes[1][hashHex][PhasePrevote]); got != 1 {
		t.Fatalf("expected duplicate vote ignored, got %d entries", got)
	}
}

func TestConsensusIgnoresReplayBlocks(t *testing.T) {
	logger := util.NewDeterministicLogger(io.Discard)
	net := NewNetwork(3000, 1, 2, 0, 0, 256, logger, 2, 4)
	nodes := buildTestNodes(net, logger, 2, 3000)

	primeNodes(nodes, 0)
	if !runUntilFinalized(net, nodes, 1, 600) {
		t.Fatalf("expected at least one finalized block")
	}
	ledger := nodes[0].LedgerSnapshot()
	blk := ledger[0]
	before := len(nodes[1].LedgerSnapshot())
	nodes[1].handleNetMsg(NetMsg{From: blk.Header.Proposer, Body: blk})
	after := len(nodes[1].LedgerSnapshot())
	if after != before {
		t.Fatalf("replayed block should not alter ledger, before=%d after=%d", before, after)
	}
}

func TestConsensusDeterministicState(t *testing.T) {
	s1 := runDeterministicScenario(7777)
	s2 := runDeterministicScenario(7777)
	if !reflect.DeepEqual(s1, s2) {
		t.Fatalf("state snapshots differ between identical runs: %v vs %v", s1, s2)
	}
}

func buildTestNodes(net *Network, logger *util.Logger, num int, seed int64) []*Node {
	nodes := make([]*Node, 0, num)
	for i := 0; i < num; i++ {
		id := NodeID(fmt.Sprintf("node%02d", i))
		n := NewNode(id, seed+int64(i), num, testChainID, logger)
		RegisterNode(net, n)
		nodes = append(nodes, n)
	}
	net.SetNodes(nodes)
	return nodes
}

func runTicks(net *Network, nodes []*Node, ticks int) {
	for i := 0; i < ticks; i++ {
		net.Tick()
		for _, n := range nodes {
			n.OnTick()
		}
	}
}

func runUntilFinalized(net *Network, nodes []*Node, height uint64, maxTicks int) bool {
	for i := 0; i < maxTicks; i++ {
		net.Tick()
		for _, n := range nodes {
			n.OnTick()
		}
		if nodes[0].FinalizedHeight() >= height {
			return true
		}
	}
	return false
}

func primeNodes(nodes []*Node, tick uint64) {
	for _, n := range nodes {
		n.maybeGenerateSelfTx(tick)
	}
}

func runDeterministicScenario(seed int64) StateSnapshot {
	logger := util.NewDeterministicLogger(io.Discard)
	net := NewNetwork(seed, 1, 2, 0.05, 0, 128, logger, 2, 4)
	nodes := buildTestNodes(net, logger, 5, seed)
	runTicks(net, nodes, 300)
	return nodes[0].SnapshotState()
}
