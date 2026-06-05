package sshw

import (
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
