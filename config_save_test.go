package sshw

import (
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
