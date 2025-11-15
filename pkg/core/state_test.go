package core

import "testing"

func TestApplyTx_ValidAndInvalid(t *testing.T) {
	state := map[string]string{}

	// valid tx
	tx := Transaction{Sender: "node00", Key: "node00/k", Value: "v1", Nonce: 1}
	if err := ApplyTx(state, tx); err != nil {
		t.Fatalf("ApplyTx failed for valid tx: %v", err)
	}
	if state["node00/k"] != "v1" {
		t.Fatalf("state not updated, got=%q want=%q", state["node00/k"], "v1")
	}

	// invalid tx (wrong key prefix)
	tx2 := Transaction{Sender: "node00", Key: "node01/k", Value: "v2", Nonce: 2}
	if err := ApplyTx(state, tx2); err == nil {
		t.Fatalf("ApplyTx should have failed for unauthorized key")
	}
}
