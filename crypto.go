package sshw

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	encPrefix = "enc:v1:argon2id:"
	argonMem  = 64 * 1024 // KiB = 64 MiB
	argonTime = 3
	argonPar  = 4
	keyLen    = 32
	saltLen   = 16
)

func NewSalt() []byte {
	s := make([]byte, saltLen)
	_, _ = rand.Read(s)
	return s
}

func deriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonTime, argonMem, argonPar, keyLen)
}

type Crypter struct {
	salt []byte
	key  []byte
}

func NewCrypter(password string, salt []byte) *Crypter {
	return &Crypter{salt: salt, key: deriveKey(password, salt)}
}

func (c *Crypter) Encrypt(plaintext string) (string, error) {
	aead, err := chacha20poly1305.NewX(c.key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := aead.Seal(nil, nonce, []byte(plaintext), nil)
	b64 := base64.RawStdEncoding.EncodeToString
	return fmt.Sprintf("%sm=%d,t=%d,p=%d:%s:%s:%s",
		encPrefix, argonMem, argonTime, argonPar,
		b64(c.salt), b64(nonce), b64(ct)), nil
}

func IsEnc(s string) bool { return strings.HasPrefix(s, "enc:v1:argon2id:") }

// SaltOf returns the salt embedded in an enc: string (for reusing the config salt).
func SaltOf(enc string) ([]byte, error) {
	_, salt, _, _, err := parseEnc(enc)
	return salt, err
}

func parseEnc(enc string) (params string, salt, nonce, ct []byte, err error) {
	if !IsEnc(enc) {
		return "", nil, nil, nil, fmt.Errorf("not an enc: value")
	}
	rest := strings.TrimPrefix(enc, encPrefix)
	parts := strings.Split(rest, ":")
	if len(parts) != 4 {
		return "", nil, nil, nil, fmt.Errorf("malformed enc: value")
	}
	dec := base64.RawStdEncoding.DecodeString
	if salt, err = dec(parts[1]); err != nil {
		return
	}
	if nonce, err = dec(parts[2]); err != nil {
		return
	}
	if ct, err = dec(parts[3]); err != nil {
		return
	}
	return parts[0], salt, nonce, ct, nil
}

// DecryptValue decrypts a self-describing enc: string with only the password.
func DecryptValue(password, enc string) (string, error) {
	_, salt, nonce, ct, err := parseEnc(enc)
	if err != nil {
		return "", err
	}
	key := deriveKey(password, salt)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}
	pt, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed (wrong master password?)")
	}
	return string(pt), nil
}
