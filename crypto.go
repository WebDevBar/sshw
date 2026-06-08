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

// deriveKeyP derives a key using explicit argon2id parameters (used by DecryptValue
// to honour the params embedded in the enc: string rather than the package constants).
func deriveKeyP(password string, salt []byte, m, t uint32, p uint8) []byte {
	return argon2.IDKey([]byte(password), salt, t, m, p, keyLen)
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
	_, salt, _, _, _, _, _, err := parseEnc(enc)
	return salt, err
}

func parseEnc(enc string) (params string, salt, nonce, ct []byte, m, t uint32, p uint8, err error) {
	if !IsEnc(enc) {
		return "", nil, nil, nil, 0, 0, 0, fmt.Errorf("not an enc: value")
	}
	rest := strings.TrimPrefix(enc, encPrefix)
	parts := strings.Split(rest, ":")
	if len(parts) != 4 {
		return "", nil, nil, nil, 0, 0, 0, fmt.Errorf("malformed enc: value")
	}
	// Parse m=<>,t=<>,p=<> from parts[0]; fall back to package constants on parse error.
	var pm, pt uint32
	var pp uint8
	_, scanErr := fmt.Sscanf(parts[0], "m=%d,t=%d,p=%d", &pm, &pt, &pp)
	if scanErr != nil {
		pm, pt, pp = argonMem, argonTime, argonPar
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
	return parts[0], salt, nonce, ct, pm, pt, pp, nil
}

// DecryptValue decrypts a self-describing enc: string with only the password.
// It uses the argon2id parameters embedded in the enc: string (m, t, p) so that
// values encrypted with non-default parameters are still decryptable.
func DecryptValue(password, enc string) (string, error) {
	_, salt, nonce, ct, m, t, p, err := parseEnc(enc)
	if err != nil {
		return "", err
	}
	key := deriveKeyP(password, salt, m, t, p)
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
