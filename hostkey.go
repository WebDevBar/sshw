package sshw

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// knownHostsCallback returns an ssh.HostKeyCallback that verifies the server's
// host key against ~/.ssh/known_hosts using trust-on-first-use (TOFU):
//   - known and matches  -> accept
//   - host not yet known  -> record the key to known_hosts, then accept (TOFU)
//   - known but CHANGED   -> refuse (possible man-in-the-middle)
//
// It fails closed (refuses) if known_hosts cannot be set up, rather than
// blindly trusting like ssh.InsecureIgnoreHostKey did.
func knownHostsCallback() ssh.HostKeyCallback {
	path, err := knownHostsPath()
	if err != nil {
		l.Errorf("host key check: cannot locate known_hosts: %v", err)
		return failClosed
	}
	checker, err := knownhosts.New(path)
	if err != nil {
		l.Errorf("host key check: cannot read known_hosts: %v", err)
		return failClosed
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := checker(hostname, remote, key); err == nil {
			return nil // known and matches
		} else {
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) {
				if len(keyErr.Want) == 0 {
					// host not in known_hosts -> trust on first use, record it
					if addErr := appendKnownHost(path, hostname, remote, key); addErr != nil {
						l.Errorf("host key check: could not record new host key: %v", addErr)
					} else {
						l.Infof("host key check: trusted new host %s (recorded to known_hosts)", hostname)
					}
					return nil
				}
				// host key changed from what we previously trusted -> refuse
				return fmt.Errorf("host key mismatch for %s: server presented a different key than the one stored in ~/.ssh/known_hosts (possible MITM) -- refusing to send credentials", hostname)
			}
			return err
		}
	}
}

// knownHostsAlgorithms returns the host-key algorithms already pinned for
// hostport in ~/.ssh/known_hosts, so the SSH client requests a key type we
// already trust (mirroring OpenSSH). Without this, Go negotiates its default
// host-key type; if the server offers a type we have NOT pinned (but the host
// IS known via another type), the known_hosts check reports a false "mismatch".
//
// x/crypto v0.52's knownhosts has no public HostKeyAlgorithms API, so we derive
// it by probing the checker with a throwaway key: for a KNOWN host the checker
// returns a *knownhosts.KeyError whose Want lists the pinned keys. Returns nil
// for unknown hosts (first-connect TOFU then uses Go's defaults).
func knownHostsAlgorithms(hostport string) []string {
	path, err := knownHostsPath()
	if err != nil {
		return nil
	}
	checker, err := knownhosts.New(path)
	if err != nil {
		return nil
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil
	}
	probeErr := checker(hostport, &net.TCPAddr{}, signer.PublicKey())
	var keyErr *knownhosts.KeyError
	if !errors.As(probeErr, &keyErr) || len(keyErr.Want) == 0 {
		return nil // unknown host (or no pin) -> let Go use its defaults
	}
	seen := map[string]bool{}
	var algos []string
	for _, kk := range keyErr.Want {
		t := kk.Key.Type()
		if seen[t] {
			continue
		}
		seen[t] = true
		if t == ssh.KeyAlgoRSA {
			// advertise the SHA-2 RSA variants too (modern servers sign with these)
			algos = append(algos, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA)
		} else {
			algos = append(algos, t)
		}
	}
	return algos
}

// pinnedCallback verifies the presented key against a pinned SHA256 fingerprint.
func pinnedCallback(fingerprint string) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if ssh.FingerprintSHA256(key) == fingerprint {
			return nil
		}
		return fmt.Errorf("host key fingerprint mismatch: server key does not match the pinned fingerprint -- refusing to connect")
	}
}

func failClosed(string, net.Addr, ssh.PublicKey) error {
	return fmt.Errorf("host key verification unavailable -- refusing to connect")
}

func knownHostsPath() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(u.HomeDir, ".ssh")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "known_hosts")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return "", err
	}
	f.Close()
	return path, nil
}

func appendKnownHost(path, hostname string, remote net.Addr, key ssh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	addrs := []string{knownhosts.Normalize(hostname)}
	if remote != nil {
		if r := knownhosts.Normalize(remote.String()); r != addrs[0] {
			addrs = append(addrs, r)
		}
	}
	_, err = f.WriteString(knownhosts.Line(addrs, key) + "\n")
	return err
}
