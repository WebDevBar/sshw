package sshw

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestNodeMarshalOmitsEmpties(t *testing.T) {
	n := &Node{Name: "db", Host: "10.0.0.5"}
	out, err := yaml.Marshal([]*Node{n})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "alias:") || strings.Contains(s, "port:") || strings.Contains(s, "password:") {
		t.Fatalf("empty fields not omitted:\n%s", s)
	}
	if !strings.Contains(s, "name: db") || !strings.Contains(s, "host: 10.0.0.5") {
		t.Fatalf("expected name+host present:\n%s", s)
	}
}

func TestNodeFingerprintRoundTrips(t *testing.T) {
	var c []*Node
	if err := yaml.Unmarshal([]byte("- {name: x, host: h, fingerprint: 'SHA256:abc'}\n"), &c); err != nil {
		t.Fatal(err)
	}
	if c[0].Fingerprint != "SHA256:abc" {
		t.Fatalf("fingerprint not parsed: %q", c[0].Fingerprint)
	}
}

func TestSaveRoundTripStripsParentAndPreservesAdvanced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sshw.yml")
	loadedPath = path
	config = []*Node{
		{Name: "-parent-"}, // synthetic; must NOT be written
		{Name: "grp", Children: []*Node{
			{Name: "-parent-"},
			{Name: "h1", Host: "1.1.1.1", Jump: []*Node{{Name: "j", Host: "9.9.9.9"}},
				CallbackShells: []*CallbackShell{{Cmd: "x"}}},
		}},
	}
	if err := Save(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "-parent-") {
		t.Fatalf("sentinel leaked into yml:\n%s", b)
	}
	// reload and confirm advanced fields survived
	loadedPath = ""
	config = nil
	data, _ := os.ReadFile(path)
	var c []*Node
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	grp := c[0]
	if grp.Name != "grp" || len(grp.Children) != 1 {
		t.Fatalf("structure wrong: %+v", grp)
	}
	h := grp.Children[0]
	if len(h.Jump) != 1 || h.Jump[0].Host != "9.9.9.9" || len(h.CallbackShells) != 1 {
		t.Fatalf("advanced fields lost: %+v", h)
	}
}
