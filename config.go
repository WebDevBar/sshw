package sshw

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kevinburke/ssh_config"
	"gopkg.in/yaml.v2"
)

type Node struct {
	Name           string           `yaml:"name,omitempty"`
	Alias          string           `yaml:"alias,omitempty"`
	Host           string           `yaml:"host,omitempty"`
	User           string           `yaml:"user,omitempty"`
	Port           int              `yaml:"port,omitempty"`
	KeyPath        string           `yaml:"keypath,omitempty"`
	AgentPath      string           `yaml:"agentpath,omitempty"`
	Passphrase     string           `yaml:"passphrase,omitempty"`
	Password       string           `yaml:"password,omitempty"`
	Fingerprint    string           `yaml:"fingerprint,omitempty"`
	CallbackShells []*CallbackShell `yaml:"callback-shells,omitempty"`
	Children       []*Node          `yaml:"children,omitempty"`
	Jump           []*Node          `yaml:"jump,omitempty"`
}

type CallbackShell struct {
	Cmd   string        `yaml:"cmd,omitempty"`
	Delay time.Duration `yaml:"delay,omitempty"`
}

func (n *Node) String() string {
	return n.Name
}

func (n *Node) user() string {
	if n.User == "" {
		return "root"
	}
	return n.User
}

func (n *Node) port() int {
	if n.Port <= 0 {
		return 22
	}
	return n.Port
}

const parentName = "-parent-"

var (
	config     []*Node
	loadedPath string

	backupDone = map[string]bool{}
	backupMu   sync.Mutex
)

func GetConfig() []*Node {
	return config
}

func LoadConfig() error {
	b, path, err := loadConfigBytesPath(".sshw", ".sshw.yml", ".sshw.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			config = nil
			loadedPath = ""
			return nil // fresh machine: empty config, reachable UI
		}
		return err
	}
	var c []*Node
	if err := yaml.Unmarshal(b, &c); err != nil {
		return err
	}
	sortNodes(c)
	config = c
	loadedPath = path
	return nil
}

// loadConfigBytesPath is LoadConfigBytes but also returns the path it read.
func loadConfigBytesPath(names ...string) ([]byte, string, error) {
	u, err := user.Current()
	if err != nil {
		return nil, "", err
	}
	var lastErr error
	for _, base := range []string{u.HomeDir, "."} {
		for _, name := range names {
			p := filepath.Join(base, name)
			b, err := os.ReadFile(p)
			if err == nil {
				return b, p, nil
			}
			lastErr = err
		}
	}
	return nil, "", lastErr
}

// sortNodes orders each level (folders and hosts together) by name,
// case-insensitively, recursing into children. Numeric folder prefixes
// (00_, 01_, ...) thus sort as intended in the picker.
func sortNodes(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	for _, n := range nodes {
		sortNodes(n.Children)
	}
}

func LoadSshConfig() error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	f, err := os.Open(filepath.Join(u.HomeDir, ".ssh/config"))
	if err != nil {
		return fmt.Errorf("open ssh config: %w", err)
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return fmt.Errorf("decode ssh config: %w", err)
	}
	var nc []*Node
	for _, host := range cfg.Hosts {
		alias := host.Patterns[0].String()
		hostName, err := cfg.Get(alias, "HostName")
		if err != nil {
			return err
		}
		if hostName != "" {
			port, _ := cfg.Get(alias, "Port")
			if port == "" {
				port = "22"
			}
			var c = new(Node)
			c.Name = alias
			c.Alias = alias
			c.Host = hostName
			c.User, _ = cfg.Get(alias, "User")
			c.Port, _ = strconv.Atoi(port)
			keyPath, _ := cfg.Get(alias, "IdentityFile")
			c.KeyPath, _ = expandHome(keyPath)
			agentPath, _ := cfg.Get(alias, "IdentityAgent")
			c.AgentPath, _ = expandHome(agentPath)
			nc = append(nc, c)
		}
	}
	config = nc
	return nil
}

func LoadConfigBytes(names ...string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	var lastErr error
	// homedir
	for i := range names {
		sshw, err := os.ReadFile(filepath.Join(u.HomeDir, names[i]))
		if err == nil {
			return sshw, nil
		}
		lastErr = err
	}
	// relative
	for i := range names {
		sshw, err := os.ReadFile(names[i])
		if err == nil {
			return sshw, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// Save writes the in-memory config back to the file it was loaded from
// (loadedPath), or ~/.sshw.yml if none was loaded. Atomic; strips synthetic
// "-parent-" nodes; writes a one-time .bak before the first save of a run.
func Save() error {
	path := loadedPath
	if path == "" {
		u, err := user.Current()
		if err != nil {
			return err
		}
		path = filepath.Join(u.HomeDir, ".sshw.yml")
		loadedPath = path
	}
	out, err := yaml.Marshal(stripParents(config))
	if err != nil {
		return err
	}
	backupOnce(path)
	return atomicWrite(path, out)
}

func stripParents(nodes []*Node) []*Node {
	out := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		if n == nil || n.Name == parentName {
			continue
		}
		cp := *n
		cp.Children = stripParents(n.Children)
		out = append(out, &cp)
	}
	return out
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".sshw-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0600); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func backupOnce(path string) {
	backupMu.Lock()
	defer backupMu.Unlock()
	if backupDone[path] {
		return
	}
	backupDone[path] = true
	if b, err := os.ReadFile(path); err == nil {
		_ = os.WriteFile(path+".bak", b, 0600)
	}
}

// SetConfig replaces the package's root node slice. Required because GetConfig()
// returns a slice value — root-level insert/delete/import in cmd/sshw produce a
// NEW root that must be written back here before Save() (which reads `config`).
func SetConfig(nodes []*Node) { config = nodes }

// expandHome expands a leading ~ in path to the user's home directory.
func expandHome(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return path, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, path[1:]), nil
}
