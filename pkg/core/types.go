package core

import (
	"crypto/sha256"
)

type NodeID string

// Transaction simple key-value set by a sender
type Transaction struct {
	Sender string
	Key    string
	Value  string
	Nonce  uint64
	Sig    []byte
}

type BlockHeader struct {
	ParentHash []byte
	Height     uint64
	StateHash  []byte
	Proposer   NodeID
	Sig        []byte
}

type Block struct {
	Header BlockHeader
	Txns   []Transaction
}

type VotePhase string

const (
	PhasePrevote   VotePhase = "PREVOTE"
	PhasePrecommit VotePhase = "PRECOMMIT"
)

type Vote struct {
	Voter     NodeID
	Height    uint64
	BlockHash []byte
	Phase     VotePhase
	Sig       []byte
}

func HashBlockHeader(h *BlockHeader) []byte {
	hb := MarshalHeaderCanonical(h)
	sum := sha256.Sum256(hb)
	return sum[:]
}
