package main

import "testing"

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
