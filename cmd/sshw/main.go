package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yinheli/sshw"
)

const prev = "-parent-"

var (
	Build  = "devel"
	V      = flag.Bool("version", false, "show version")
	H      = flag.Bool("help", false, "show help")
	S      = flag.Bool("s", false, "use local ssh config '~/.ssh/config'")
	CopyID = flag.Bool("copy-id", false, "copy SSH public key to selected host")

	log      = sshw.GetLogger()
	settings *sshw.Settings
)

// runShare copies the node's host details (and safe ssh line) to the clipboard.
// If the node's secrets are enc:-encoded, they are decrypted on a COPY of the
// node so the shared config tree is never mutated.
func runShare(n *sshw.Node) {
	cp, err := decryptedClone(n)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  share: decrypt error: %v\n", err)
		return
	}
	txt := sshw.ShareText(cp)
	toolOK, _, copyErr := sshw.Copy(txt, func(s string) { fmt.Fprint(os.Stderr, s) })
	if copyErr != nil {
		fmt.Fprintf(os.Stderr, "  share: clipboard error: %v\n", copyErr)
		return
	}
	if toolOK {
		fmt.Fprint(os.Stderr, "  ✓ copied to clipboard\n")
	}
	fmt.Fprint(os.Stderr, "  ⚠ credentials copied to clipboard\n")
}

func findAlias(nodes []*sshw.Node, nodeAlias string) *sshw.Node {
	for _, node := range nodes {
		if node.Alias == nodeAlias {
			return node
		}
		if len(node.Children) > 0 {
			if result := findAlias(node.Children, nodeAlias); result != nil {
				return result
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		return
	}

	if *H {
		flag.Usage()
		return
	}

	if *V {
		fmt.Println("sshw - ssh client wrapper for automatic login")
		fmt.Println("  git version:", Build)
		fmt.Println("  go version :", runtime.Version())
		return
	}
	if *S {
		err := sshw.LoadSshConfig()
		if err != nil {
			log.Error("load ssh config error", err)
			os.Exit(1)
		}
	} else {
		err := sshw.LoadConfig()
		if err != nil {
			log.Error("load config error", err)
			os.Exit(1)
		}
	}

	settings, _ = sshw.LoadSettings()
	if settings == nil {
		settings = &sshw.Settings{}
	}
	shareEnabled = settings.Share.Enabled

	// login by alias
	if len(os.Args) > 1 {
		var nodeAlias = os.Args[1]
		var nodes = sshw.GetConfig()
		var node = findAlias(nodes, nodeAlias)
		if node != nil {
			dn, err := decryptedClone(node)
			if err != nil {
				log.Error("decrypt:", err)
				os.Exit(1)
			}
			client := sshw.NewClient(dn)
			client.Login()
			return
		}
	}

	leaves := flattenLeaves(sshw.GetConfig(), "")
	node := choose(nil, sshw.GetConfig(), leaves, "")
	if node == nil {
		return
	}

	if *CopyID {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Error("cannot find home directory:", err)
			os.Exit(1)
		}
		pubKey, err := os.ReadFile(filepath.Join(home, ".ssh", "id_ed25519.pub"))
		if err != nil {
			pubKey, err = os.ReadFile(filepath.Join(home, ".ssh", "id_rsa.pub"))
			if err != nil {
				log.Error("no public key found (~/.ssh/id_ed25519.pub or ~/.ssh/id_rsa.pub)")
				os.Exit(1)
			}
		}
		dn, err := decryptedClone(node)
		if err != nil {
			log.Error("decrypt:", err)
			os.Exit(1)
		}
		client := sshw.NewClient(dn)
		if err := client.CopyID(pubKey); err != nil {
			log.Error("copy-id failed:", err)
			os.Exit(1)
		}
		user := node.User
		if user == "" {
			user = "root"
		}
		fmt.Printf("public key copied to %s@%s\n", user, node.Host)
		return
	}

	dn, err := decryptedClone(node)
	if err != nil {
		log.Error("decrypt:", err)
		os.Exit(1)
	}
	client := sshw.NewClient(dn)
	client.Login()
}

// choose drives the picker loop. parent is the level to return to on Esc/up,
// trees is the current level's nodes, leaves is the global search index, and
// levelPath is the "/"-joined breadcrumb for the current level.
// Returns the selected connectable node, or nil to quit.
func choose(parent, trees []*sshw.Node, leaves []leaf, levelPath string) *sshw.Node {
	for {
		node, path, action, err := selectNode("select host", trees, leaves, 20, levelPath)
		if err != nil {
			return nil
		}

		switch action {
		case "quit":
			return nil

		case "nav":
			// Esc or Backspace while in a folder — go up one level.
			if node != nil && node.Name == prev {
				if parent == nil {
					return choose(nil, sshw.GetConfig(), flattenLeaves(sshw.GetConfig(), ""), "")
				}
				return choose(nil, parent, flattenLeaves(sshw.GetConfig(), ""), "")
			}
			// Esc at root -> quit
			return nil

		case "connect":
			if node == nil {
				continue
			}
			// Drill into folder
			if len(node.Children) > 0 {
				first := node.Children[0]
				if first.Name != prev {
					first = &sshw.Node{Name: prev}
					node.Children = append(node.Children[:0], append([]*sshw.Node{first}, node.Children...)...)
				}
				childPath := node.Name
				if levelPath != "" {
					childPath = levelPath + "/" + node.Name
				}
				result := choose(trees, node.Children, leaves, childPath)
				if result != nil {
					return result
				}
				// Returned nil from sub-level (quit) — propagate
				return nil
			}
			// It's a -parent- sentinel -> go up
			if node.Name == prev {
				if parent == nil {
					return choose(nil, sshw.GetConfig(), flattenLeaves(sshw.GetConfig(), ""), "")
				}
				return choose(nil, parent, flattenLeaves(sshw.GetConfig(), ""), "")
			}
			return node

		case "add":
			f := newFormModel(nil, path)
			if runForm(f) {
				n, folder, ferr := f.toNode()
				if ferr == nil {
					if eerr := encryptIfEnabled(n); eerr == nil {
						root := sshw.InsertNode(sshw.GetConfig(), folder, n)
						sshw.SetConfig(root)
						_ = sshw.Save()
					} else {
						log.Error("encrypt failed; host not saved:", eerr)
					}
				} else {
					log.Error("invalid host entry; host not saved:", ferr)
				}
			}
			// Refresh and re-enter picker
			trees = sshw.GetConfig()
			leaves = flattenLeaves(sshw.GetConfig(), "")

		case "edit":
			// No-op on folder rows or empty list
			if node == nil || node.Name == prev || len(node.Children) > 0 {
				continue
			}
			f := newFormModel(node, path)
			if runForm(f) {
				n, folder, ferr := f.toNode()
				if ferr == nil {
					if eerr := encryptIfEnabled(n); eerr == nil {
						root := sshw.GetConfig()
						// Remove old node from its original path
						root, _ = sshw.DeleteNode(root, f.origPath, node.Name)
						// Insert updated node at (possibly new) folder
						root = sshw.InsertNode(root, folder, n)
						sshw.SetConfig(root)
						_ = sshw.Save()
					} else {
						log.Error("encrypt failed; host not saved:", eerr)
					}
				} else {
					log.Error("invalid host entry; host not saved:", ferr)
				}
			}
			trees = sshw.GetConfig()
			leaves = flattenLeaves(sshw.GetConfig(), "")

		case "delete":
			if node == nil || node.Name == prev {
				continue
			}
			// Inline confirm prompt
			if confirmDelete(node) {
				root, _ := sshw.DeleteNode(sshw.GetConfig(), path, node.Name)
				sshw.SetConfig(root)
				_ = sshw.Save()
			}
			trees = sshw.GetConfig()
			leaves = flattenLeaves(sshw.GetConfig(), "")

		case "options":
			runOptions()
			trees = sshw.GetConfig()
			leaves = flattenLeaves(sshw.GetConfig(), "")

		case "share":
			if node != nil && node.Host != "" {
				runShare(node)
			}
		}
	}
}

// confirmDelete renders a [y/N] prompt and returns true if the user presses y or Y.
func confirmDelete(node *sshw.Node) bool {
	user := node.User
	if user == "" {
		user = "root"
	}
	fmt.Fprintf(os.Stderr, "  Delete %q (%s@%s)? [y/N] ", node.Name, user, node.Host)
	buf := make([]byte, 1)
	n, _ := os.Stdin.Read(buf)
	fmt.Fprintln(os.Stderr)
	return n == 1 && (buf[0] == 'y' || buf[0] == 'Y')
}

// encryptIfEnabled encrypts the secrets on n in-place when master password is
// enabled. Called after f.toNode() in add and edit handlers (Task 12 Step 4).
func encryptIfEnabled(n *sshw.Node) error {
	if settings == nil || !settings.MasterPassword.Enabled {
		return nil
	}
	pw, err := ensureMaster(sshw.GetConfig(), settings.MasterPassword.Verifier, true)
	if err != nil {
		return err
	}
	salt := sshw.OperativeSalt(sshw.GetConfig(), settings.MasterPassword.Verifier)
	return sshw.EncryptAll([]*sshw.Node{n}, pw, salt)
}

// decryptedClone returns a connect-ready CLONE of node with its (and its used jump
// hop's) secrets decrypted. Never mutates the shared config tree.
func decryptedClone(node *sshw.Node) (*sshw.Node, error) {
	cp := *node
	if len(node.Jump) > 0 {
		cp.Jump = make([]*sshw.Node, len(node.Jump))
		for i, j := range node.Jump {
			jc := *j
			cp.Jump[i] = &jc
		}
	}
	enc := sshw.IsEnc(cp.Password) || sshw.IsEnc(cp.Passphrase) ||
		(len(cp.Jump) > 0 && (sshw.IsEnc(cp.Jump[0].Password) || sshw.IsEnc(cp.Jump[0].Passphrase)))
	if !enc {
		return &cp, nil
	}
	pw, err := ensureMaster(sshw.GetConfig(), settings.MasterPassword.Verifier, settings.MasterPassword.Enabled)
	if err != nil {
		return nil, err
	}
	if err := sshw.DecryptNode(&cp, pw); err != nil {
		return nil, err
	}
	if len(cp.Jump) > 0 {
		if err := sshw.DecryptNode(cp.Jump[0], pw); err != nil {
			return nil, err
		}
	}
	return &cp, nil
}
