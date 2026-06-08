package sshw

import (
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Settings struct {
	MasterPassword struct {
		Enabled  bool   `yaml:"enabled"`
		Verifier string `yaml:"verifier,omitempty"`
	} `yaml:"master_password"`
	Share struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"share"`
}

func settingsPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		base = filepath.Join(u.HomeDir, ".config")
	}
	return filepath.Join(base, "sshw", "settings.yml"), nil
}

func LoadSettings() (*Settings, error) {
	p, err := settingsPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil
		}
		return nil, err
	}
	var s Settings
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveSettings(s *Settings) error {
	p, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	out, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return atomicWrite(p, out) // reuse config.go's atomicWrite
}
