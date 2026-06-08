package sshw

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	salt := NewSalt()
	c := NewCrypter("hunter2", salt)
	enc, err := c.Encrypt("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	if !IsEnc(enc) || !strings.HasPrefix(enc, "enc:v1:argon2id:") {
		t.Fatalf("bad enc string: %q", enc)
	}
	pt, err := DecryptValue("hunter2", enc)
	if err != nil {
		t.Fatal(err)
	}
	if pt != "s3cr3t" {
		t.Fatalf("round-trip mismatch: %q", pt)
	}
}

func TestDecryptWrongPasswordFails(t *testing.T) {
	c := NewCrypter("right", NewSalt())
	enc, _ := c.Encrypt("x")
	if _, err := DecryptValue("wrong", enc); err == nil {
		t.Fatal("expected auth failure on wrong password")
	}
}

func TestSaltEmbeddedSoStandalone(t *testing.T) {
	// a value encrypted under one Crypter is decryptable with only the password
	c := NewCrypter("pw", NewSalt())
	enc, _ := c.Encrypt("v")
	if _, err := DecryptValue("pw", enc); err != nil {
		t.Fatalf("standalone decrypt failed: %v", err)
	}
}
