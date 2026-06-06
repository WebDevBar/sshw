package main

import (
	"testing"

	"github.com/yinheli/sshw"
)

func TestMatchText(t *testing.T) {
	cases := []struct {
		input, content string
		want           bool
	}{
		{"wsh", "_ASHL/_WS2.0_DEV _WSHUB wshub 52.3.27.52", true},
		{"WSH", "x _wshub y", true},
		{"dev hub", "_ASHL/_WS2.0_DEV _WSHUB wshub 1.2.3.4", true},
		{"dev zzz", "_ASHL/_WS2.0_DEV _WSHUB wshub 1.2.3.4", false},
		{"", "anything", true},
	}
	for _, c := range cases {
		if got := matchText(c.input, c.content); got != c.want {
			t.Errorf("matchText(%q,%q)=%v want %v", c.input, c.content, got, c.want)
		}
	}
}

func TestFlattenLeaves(t *testing.T) {
	cfg := []*sshw.Node{
		{Name: "top", Host: "10.0.0.1", User: "root"},
		{Name: "GROUP", Children: []*sshw.Node{
			{Name: "a", Host: "10.0.0.2"},
			{Name: "SUB", Children: []*sshw.Node{
				{Name: "b", Host: "10.0.0.3"},
			}},
			{Name: "placeholder"},
			{Name: prev},
		}},
	}
	got := flattenLeaves(cfg, "")
	want := map[string]string{
		"top": "", "a": "GROUP", "b": "GROUP/SUB",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d leaves, want %d: %+v", len(got), len(want), got)
	}
	for _, l := range got {
		if want[l.node.Name] != l.path {
			t.Errorf("leaf %q path=%q want %q", l.node.Name, l.path, want[l.node.Name])
		}
	}
}

func TestLeafContent(t *testing.T) {
	l := leaf{node: &sshw.Node{Name: "WSHUB", User: "wshub", Host: "1.2.3.4"}, path: "A/B"}
	if got := leafContent(l); got != "A/B WSHUB wshub 1.2.3.4" {
		t.Errorf("leafContent=%q", got)
	}
}
