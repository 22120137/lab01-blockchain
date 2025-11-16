package core

import (
	"io"
	"testing"

	"lab01/pkg/util"
)

func TestNodeAddTxSequential(t *testing.T) {
	logger := util.NewDeterministicLogger(io.Discard)
	chainID := "test-chain"
	n := NewNode(NodeID("node00"), 1, 4, chainID, logger)
	n.pubs[n.id] = n.kp.Pub

	domain := "TX:" + chainID
	tx1 := Transaction{Sender: string(n.id), Key: string(n.id) + "/k", Value: "v1", Nonce: 1}
	tx1.Sig = SignWithDomain(domain, MarshalTxCanonical(tx1), n.kp.Priv)
	if !n.addTxToPool(tx1) {
		t.Fatalf("expected first tx accepted")
	}

	// duplicate nonce rejected
	txDup := Transaction{Sender: string(n.id), Key: string(n.id) + "/kdup", Value: "dup", Nonce: 1}
	txDup.Sig = SignWithDomain(domain, MarshalTxCanonical(txDup), n.kp.Priv)
	if n.addTxToPool(txDup) {
		t.Fatalf("duplicate nonce should be rejected")
	}

	// next nonce accepted
	tx2 := Transaction{Sender: string(n.id), Key: string(n.id) + "/k2", Value: "v2", Nonce: 2}
	tx2.Sig = SignWithDomain(domain, MarshalTxCanonical(tx2), n.kp.Priv)
	if !n.addTxToPool(tx2) {
		t.Fatalf("expected second tx accepted")
	}
}
