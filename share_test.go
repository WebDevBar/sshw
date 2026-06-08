package sshw

import (
	"strings"
	"testing"
)

func TestShareText(t *testing.T) {
	n := &Node{Name: "db", Host: "10.0.0.5", User: "root", Port: 2222, Password: "pw", KeyPath: "~/.ssh/id"}
	out := ShareText(n)
	for _, want := range []string{"name:     db", "host:     10.0.0.5", "password: pw",
		`ssh -p 2222 -i '~/.ssh/id' -- 'root'@'10.0.0.5'`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestShareSSHLineSafety(t *testing.T) {
	// host beginning with "-" must NOT yield an ssh option; command refused
	n := &Node{Name: "x", Host: "-oProxyCommand=evil", User: "root"}
	out := ShareText(n)
	if strings.Contains(out, "ssh ") {
		t.Fatalf("unsafe host should suppress the ssh line:\n%s", out)
	}
	// metacharacters get shell-quoted
	n2 := &Node{Name: "y", Host: "h o;st", User: "ro ot", Port: 22}
	out2 := ShareText(n2)
	if !strings.Contains(out2, "'ro ot'@'h o;st'") && !strings.Contains(out2, "-- 'ro ot'@'h o;st'") {
		t.Fatalf("expected shell-quoted destination:\n%s", out2)
	}
}
