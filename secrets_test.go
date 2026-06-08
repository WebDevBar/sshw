package sshw

import "testing"

func encTree() []*Node {
	return []*Node{
		{Name: "a", Host: "h", Password: "p1", Passphrase: "pp1",
			Jump: []*Node{{Name: "j", Host: "jh", Password: "jp"}}},
		{Name: "grp", Children: []*Node{{Name: "b", Host: "h2", Password: "p2"}}},
	}
}

func TestEncryptAllThenDecryptAll(t *testing.T) {
	nodes := encTree()
	salt := NewSalt()
	if err := EncryptAll(nodes, "mp", salt); err != nil {
		t.Fatal(err)
	}
	if !IsEnc(nodes[0].Password) || !IsEnc(nodes[0].Passphrase) || !IsEnc(nodes[0].Jump[0].Password) {
		t.Fatal("not all secrets encrypted (incl jump)")
	}
	if !AnyEncrypted(nodes) {
		t.Fatal("AnyEncrypted should be true")
	}
	if err := DecryptAll(nodes, "mp"); err != nil {
		t.Fatal(err)
	}
	if nodes[0].Password != "p1" || nodes[0].Jump[0].Password != "jp" || nodes[1].Children[0].Password != "p2" {
		t.Fatal("decrypt-all round-trip failed")
	}
}

func TestDecryptAllWrongPassword(t *testing.T) {
	nodes := encTree()
	_ = EncryptAll(nodes, "mp", NewSalt())
	if err := DecryptAll(nodes, "nope"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestVerifier(t *testing.T) {
	salt := NewSalt()
	v, _ := MakeVerifier("mp", salt)
	if !CheckVerifier("mp", v) {
		t.Fatal("verifier should accept correct password")
	}
	if CheckVerifier("bad", v) {
		t.Fatal("verifier should reject wrong password")
	}
}
