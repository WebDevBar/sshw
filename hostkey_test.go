package sshw

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestPinnedFingerprintMatch(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(priv)
	pub := signer.PublicKey()
	fp := ssh.FingerprintSHA256(pub)
	cb := pinnedCallback(fp) // refuses on mismatch, accepts on match
	if err := cb("host:22", nil, pub); err != nil {
		t.Fatalf("match should pass: %v", err)
	}
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	signer2, _ := ssh.NewSignerFromKey(priv2)
	if err := cb("host:22", nil, signer2.PublicKey()); err == nil {
		t.Fatal("mismatch must be refused")
	}
}
