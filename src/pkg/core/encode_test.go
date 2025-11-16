package core

import "testing"

func TestStateHashDeterministic(t *testing.T) {
	s1 := NewState()
	s1.Data["b"] = "2"
	s1.Data["a"] = "1"
	s1.Nonces["alice"] = 3

	s2 := NewState()
	s2.Data["a"] = "1"
	s2.Data["b"] = "2"
	s2.Nonces["alice"] = 3

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
