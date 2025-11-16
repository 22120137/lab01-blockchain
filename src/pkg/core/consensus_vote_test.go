package core

import (
	"io"
	"testing"

	"lab01/pkg/util"
)

func TestHandleVoteRejectsBadSignature(t *testing.T) {
	logger := util.NewDeterministicLogger(io.Discard)
	n := NewNode(NodeID("node00"), 1, 4, "test-chain", logger)
	kp := GenKeypair()
	n.pubs[NodeID("node01")] = kp.Pub

	// craft vote with wrong signature bytes (all zeros)
	v := Vote{
		Voter:     NodeID("node01"),
		Height:    1,
		BlockHash: []byte("hash"),
		Phase:     PhasePrevote,
		Sig:       []byte("bad"),
	}
	n.handleNetMsg(NetMsg{From: NodeID("node01"), To: n.id, Body: v})
	if len(n.votes[1]) != 0 {
		t.Fatalf("expected vote not to be tallied when signature invalid")
	}
}
