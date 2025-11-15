package core

import (
	"errors"
)

func ApplyTx(state map[string]string, tx Transaction) error {
	// verify signature was done externally
	// simple ownership rule: key must start with sender+/
	if len(tx.Sender) == 0 { return errors.New("empty sender") }
	if len(tx.Key) == 0 { return errors.New("empty key") }
	prefix := tx.Sender + "/"
	if len(tx.Key) < len(prefix) || tx.Key[:len(prefix)] != prefix {
		return errors.New("not owner")
	}
	state[tx.Key] = tx.Value
	return nil
}
