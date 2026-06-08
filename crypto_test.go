package sshw

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
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

func TestDecryptHonorsEmbeddedParams(t *testing.T) {
	// Encrypt with non-default params (smaller/faster for testing).
	// We build the enc: string manually using the same format as Crypter.Encrypt
	// but with m=8192, t=1, p=1.
	const (
		testM uint32 = 8192
		testT uint32 = 1
		testP uint8  = 1
	)
	const plaintext = "secret-value"
	const password = "test-password"

	salt := NewSalt()

	// Derive key with non-default params and encrypt.
	key := deriveKeyP(password, salt, testM, testT, testP)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	ct := aead.Seal(nil, nonce, []byte(plaintext), nil)

	b64 := base64.RawStdEncoding.EncodeToString
	enc := fmt.Sprintf("%sm=%d,t=%d,p=%d:%s:%s:%s",
		encPrefix, testM, testT, testP,
		b64(salt), b64(nonce), b64(ct))

	// DecryptValue must succeed by reading the embedded params.
	got, err := DecryptValue(password, enc)
	if err != nil {
		t.Fatalf("DecryptValue with embedded non-default params failed: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round-trip mismatch: got %q, want %q", got, plaintext)
	}

	// Sanity-check: if we naively used the package constants (argonMem, argonTime, argonPar)
	// instead of the embedded params, the key would be wrong and decryption would fail.
	// Demonstrate this explicitly.
	wrongKey := deriveKey(password, salt) // uses package-constant params
	wrongAead, _ := chacha20poly1305.NewX(wrongKey)
	if _, wrongErr := wrongAead.Open(nil, nonce, ct, nil); wrongErr == nil {
		t.Fatal("expected decryption to fail with default (wrong) params, but it succeeded — " +
			"test-only params must differ from package constants")
	}
}
