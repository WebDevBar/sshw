package sshw

const verifierToken = "sshw-verify"

// walkSecrets calls fn for the address of each Password/Passphrase in the tree
// (including children and jump nodes).
func walkSecrets(nodes []*Node, fn func(*string)) {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		fn(&n.Password)
		fn(&n.Passphrase)
		walkSecrets(n.Children, fn)
		walkSecrets(n.Jump, fn)
	}
}

func AnyEncrypted(nodes []*Node) bool {
	found := false
	walkSecrets(nodes, func(s *string) {
		if IsEnc(*s) {
			found = true
		}
	})
	return found
}

// firstEnc returns any encrypted value in the tree (for salt + validation).
func firstEnc(nodes []*Node) string {
	var out string
	walkSecrets(nodes, func(s *string) {
		if out == "" && IsEnc(*s) {
			out = *s
		}
	})
	return out
}

func EncryptAll(nodes []*Node, password string, salt []byte) error {
	c := NewCrypter(password, salt)
	var err error
	walkSecrets(nodes, func(s *string) {
		if err != nil || *s == "" || IsEnc(*s) {
			return
		}
		enc, e := c.Encrypt(*s)
		if e != nil {
			err = e
			return
		}
		*s = enc
	})
	return err
}

func DecryptAll(nodes []*Node, password string) error {
	var err error
	walkSecrets(nodes, func(s *string) {
		if err != nil || !IsEnc(*s) {
			return
		}
		pt, e := DecryptValue(password, *s)
		if e != nil {
			err = e
			return
		}
		*s = pt
	})
	return err
}

// DecryptNode decrypts the secrets on a single node in place (for connect/share).
func DecryptNode(n *Node, password string) error {
	if n == nil {
		return nil
	}
	for _, s := range []*string{&n.Password, &n.Passphrase} {
		if IsEnc(*s) {
			pt, err := DecryptValue(password, *s)
			if err != nil {
				return err
			}
			*s = pt
		}
	}
	return nil
}

func MakeVerifier(password string, salt []byte) (string, error) {
	return NewCrypter(password, salt).Encrypt(verifierToken)
}

func CheckVerifier(password, verifier string) bool {
	pt, err := DecryptValue(password, verifier)
	return err == nil && pt == verifierToken
}

// OperativeSalt returns the salt to use for new encryptions: from a real enc:
// value if any, else from the verifier, else a fresh salt.
func OperativeSalt(nodes []*Node, verifier string) []byte {
	if e := firstEnc(nodes); e != "" {
		if s, err := SaltOf(e); err == nil {
			return s
		}
	}
	if verifier != "" {
		if s, err := SaltOf(verifier); err == nil {
			return s
		}
	}
	return NewSalt()
}

// ValidatePassword implements the spec authority rule: host enc: secrets win;
// the verifier is used only when there are zero host secrets.
func ValidatePassword(nodes []*Node, verifier, password string) bool {
	if e := firstEnc(nodes); e != "" {
		_, err := DecryptValue(password, e)
		return err == nil
	}
	if verifier != "" {
		return CheckVerifier(password, verifier)
	}
	return false
}
