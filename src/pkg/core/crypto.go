package core

import (
	"crypto/ed25519"
	"crypto/rand"
)

type Keypair struct {
	Priv ed25519.PrivateKey
	Pub  ed25519.PublicKey
}

func GenKeypair() Keypair {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	return Keypair{Priv: priv, Pub: pub}
}

func SignWithDomain(domain string, msg []byte, priv ed25519.PrivateKey) []byte {
	b := append([]byte(domain), msg...)
	return ed25519.Sign(priv, b)
}

func VerifyWithDomain(domain string, msg, sig []byte, pub ed25519.PublicKey) bool {
	b := append([]byte(domain), msg...)
	return ed25519.Verify(pub, b, sig)
}
