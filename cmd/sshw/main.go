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

	log = sshw.GetLogger()
)

// runShare is a stub — implemented in Task 19.
func runShare(n *sshw.Node) {}

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

	// login by alias
	if len(os.Args) > 1 {
		var nodeAlias = os.Args[1]
		var nodes = sshw.GetConfig()
		var node = findAlias(nodes, nodeAlias)
		if node != nil {
			client := sshw.NewClient(node)
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
		client := sshw.NewClient(node)
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

	client := sshw.NewClient(node)
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
					root := sshw.InsertNode(sshw.GetConfig(), folder, n)
					sshw.SetConfig(root)
					_ = sshw.Save()
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
					root := sshw.GetConfig()
					// Remove old node from its original path
					root, _ = sshw.DeleteNode(root, f.origPath, node.Name)
					// Insert updated node at (possibly new) folder
					root = sshw.InsertNode(root, folder, n)
					sshw.SetConfig(root)
					_ = sshw.Save()
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
