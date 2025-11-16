package tests

import (
	"testing"

	"lab01/pkg/core"
)

func TestSignVerify(t *testing.T) {
	kp := core.GenKeypair()
	msg := []byte("hello")
	sig := core.SignWithDomain("DOM:", msg, kp.Priv)
	if !core.VerifyWithDomain("DOM:", msg, sig, kp.Pub) {
		t.Fatal("verify failed")
	}
	if core.VerifyWithDomain("WRONG:", msg, sig, kp.Pub) {
		t.Fatal("domain separation failed")
	}
}
