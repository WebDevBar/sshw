package sshw

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	s := &Settings{}
	s.MasterPassword.Enabled = true
	s.MasterPassword.Verifier = "enc:v1:..."
	s.Share.Enabled = true
	if err := SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sshw", "settings.yml")); err != nil {
		t.Fatalf("settings file not written: %v", err)
	}
	got, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !got.MasterPassword.Enabled || got.MasterPassword.Verifier != "enc:v1:..." || !got.Share.Enabled {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestLoadSettingsAbsentIsDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if s.MasterPassword.Enabled || s.Share.Enabled {
		t.Fatal("absent settings should default to all-off")
	}
}
