package core

import "testing"

func TestApplyTx_ValidAndInvalid(t *testing.T) {
	state := NewState()

	// valid tx
	tx := Transaction{Sender: "node00", Key: "node00/k", Value: "v1", Nonce: 1}
	if err := ApplyTx(state, tx); err != nil {
		t.Fatalf("ApplyTx failed for valid tx: %v", err)
	}
	if state.Data["node00/k"] != "v1" {
		t.Fatalf("state not updated, got=%q want=%q", state.Data["node00/k"], "v1")
	}
	if state.Nonces["node00"] != 1 {
		t.Fatalf("nonce not updated, got=%d want=1", state.Nonces["node00"])
	}

	// invalid tx (wrong key prefix)
	tx2 := Transaction{Sender: "node00", Key: "node01/k", Value: "v2", Nonce: 2}
	if err := ApplyTx(state, tx2); err == nil {
		t.Fatalf("ApplyTx should have failed for unauthorized key")
	}

	// replay nonce
	tx3 := Transaction{Sender: "node00", Key: "node00/z", Value: "v3", Nonce: 1}
	if err := ApplyTx(state, tx3); err == nil {
		t.Fatalf("ApplyTx should have failed due to nonce replay")
	}

	// next correct nonce should work
	tx4 := Transaction{Sender: "node00", Key: "node00/z", Value: "v4", Nonce: 2}
	if err := ApplyTx(state, tx4); err != nil {
		t.Fatalf("expected valid nonce: %v", err)
	}
}
