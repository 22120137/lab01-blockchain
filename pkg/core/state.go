package core

import (
	"errors"
)

// State stores deterministic KV values and per-sender nonce counters.
type State struct {
	Data   map[string]string
	Nonces map[string]uint64
}

func NewState() *State {
	return &State{
		Data:   make(map[string]string),
		Nonces: make(map[string]uint64),
	}
}

func (s *State) Clone() *State {
	copy := NewState()
	for k, v := range s.Data {
		copy.Data[k] = v
	}
	for sender, nonce := range s.Nonces {
		copy.Nonces[sender] = nonce
	}
	return copy
}

func ApplyTx(state *State, tx Transaction) error {
	if len(tx.Sender) == 0 {
		return errors.New("empty sender")
	}
	if len(tx.Key) == 0 {
		return errors.New("empty key")
	}
	prefix := tx.Sender + "/"
	if len(tx.Key) < len(prefix) || tx.Key[:len(prefix)] != prefix {
		return errors.New("not owner")
	}
	expected := state.Nonces[tx.Sender] + 1
	if tx.Nonce != expected {
		return errors.New("bad nonce")
	}
	state.Nonces[tx.Sender] = tx.Nonce
	state.Data[tx.Key] = tx.Value
	return nil
}
