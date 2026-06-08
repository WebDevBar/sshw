package sshw

import "testing"

func TestClipboardCommandForGOOS(t *testing.T) {
	if cmd := clipboardCmd("darwin", func(string) bool { return true }); len(cmd) == 0 || cmd[0] != "pbcopy" {
		t.Fatalf("darwin should use pbcopy, got %v", cmd)
	}
	// linux: prefer wl-copy if present
	got := clipboardCmd("linux", func(b string) bool { return b == "wl-copy" })
	if len(got) == 0 || got[0] != "wl-copy" {
		t.Fatalf("linux wl-copy preferred, got %v", got)
	}
	// linux: fall to xclip
	got = clipboardCmd("linux", func(b string) bool { return b == "xclip" })
	if len(got) == 0 || got[0] != "xclip" {
		t.Fatalf("linux xclip fallback, got %v", got)
	}
	// nothing available -> nil (caller uses OSC 52)
	if clipboardCmd("linux", func(string) bool { return false }) != nil {
		t.Fatal("no tool -> nil")
	}
}

func TestOSC52(t *testing.T) {
	s := osc52("hi")
	if len(s) == 0 || s[:5] != "\x1b]52;" {
		t.Fatalf("bad osc52: %q", s)
	}
}
