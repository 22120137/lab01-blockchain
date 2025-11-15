package core

import "testing"

func TestStateHashDeterministic(t *testing.T) {
	s1 := map[string]string{"b": "2", "a": "1"}
	s2 := map[string]string{"a": "1", "b": "2"}

	h1 := StateHash(s1)
	h2 := StateHash(s2)
	if string(h1) != string(h2) {
		t.Fatalf("StateHash not deterministic: %x vs %x", h1, h2)
	}
}

func TestMarshalHeaderCanonicalStable(t *testing.T) {
	h := &BlockHeader{
		ParentHash: []byte("parent"),
		Height:     5,
		StateHash:  []byte("state"),
		Proposer:   NodeID("node00"),
	}
	b1 := MarshalHeaderCanonical(h)
	b2 := MarshalHeaderCanonical(h)
	if string(b1) != string(b2) {
		t.Fatalf("MarshalHeaderCanonical not stable")
	}
}
