package core

import "testing"

func TestSignVerifyDomainSeparation(t *testing.T) {
	kp := GenKeypair()
	msg := []byte("hello")
	sig := SignWithDomain("DOM:", msg, kp.Priv)
	if !VerifyWithDomain("DOM:", msg, sig, kp.Pub) {
		t.Fatal("verify should succeed with same domain")
	}
	if VerifyWithDomain("OTHER:", msg, sig, kp.Pub) {
		t.Fatal("verify should fail with different domain")
	}
}
