package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/yinheli/sshw"
	"golang.org/x/term"
)

var cachedMasterPassword string // empty = not yet entered this run

// ensureMaster returns the validated master password for the current config,
// prompting (once per run) if needed. nodes+verifier define what to validate
// against (authority rule). Returns "" if encryption isn't enabled.
func ensureMaster(nodes []*sshw.Node, verifier string, enabled bool) (string, error) {
	if !enabled && !sshw.AnyEncrypted(nodes) {
		return "", nil
	}
	if cachedMasterPassword != "" {
		return cachedMasterPassword, nil
	}
	for attempts := 0; attempts < 3; attempts++ {
		fmt.Fprint(os.Stderr, "🔒 master password: ")
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		pw := string(b)
		if sshw.ValidatePassword(nodes, verifier, pw) {
			cachedMasterPassword = pw
			return pw, nil
		}
		fmt.Fprintln(os.Stderr, "  wrong master password")
	}
	return "", fmt.Errorf("master password not provided")
}
