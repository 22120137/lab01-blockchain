package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"sort"
)

func writeVarLenBytes(buf *bytes.Buffer, b []byte) {
	binary.Write(buf, binary.BigEndian, uint32(len(b)))
	buf.Write(b)
}
func writeVarLenString(buf *bytes.Buffer, s string) { writeVarLenBytes(buf, []byte(s)) }

func MarshalTxCanonical(tx Transaction) []byte {
	buf := bytes.NewBuffer(nil)
	writeVarLenString(buf, tx.Sender)
	writeVarLenString(buf, tx.Key)
	writeVarLenString(buf, tx.Value)
	binary.Write(buf, binary.BigEndian, tx.Nonce)
	return buf.Bytes()
}

func MarshalHeaderCanonical(h *BlockHeader) []byte {
	buf := bytes.NewBuffer(nil)
	writeVarLenBytes(buf, h.ParentHash)
	binary.Write(buf, binary.BigEndian, h.Height)
	writeVarLenBytes(buf, h.StateHash)
	writeVarLenString(buf, string(h.Proposer))
	return buf.Bytes()
}

func MarshalVoteCanonical(v *Vote) []byte {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.BigEndian, v.Height)
	writeVarLenBytes(buf, v.BlockHash)
	writeVarLenString(buf, string(v.Voter))
	writeVarLenString(buf, string(v.Phase))
	return buf.Bytes()
}

func StateHash(state map[string]string) []byte {
	buf := bytes.NewBuffer(nil)
	keys := make([]string, 0, len(state))
	for k := range state {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		writeVarLenString(buf, k)
		writeVarLenString(buf, state[k])
	}
	sum := sha256.Sum256(buf.Bytes())
	return sum[:]
}

