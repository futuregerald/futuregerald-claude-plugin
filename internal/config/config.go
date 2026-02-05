package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the .skill-installer.yaml configuration file.
type Config struct {
	Target       string   `yaml:"target"`
	Tags         []string `yaml:"tags"`
	Languages    []string `yaml:"languages"`
	SkipClaudeMD bool     `yaml:"skip_claude_md"`
	From         string   `yaml:"from"`
}

// DefaultConfigFiles are the filenames to look for.
var DefaultConfigFiles = []string{
	".skill-installer.yaml",
	".skill-installer.yml",
	"skill-installer.yaml",
	"skill-installer.yml",
}

// Load attempts to load configuration from the current directory.
func Load(dir string) (*Config, error) {
	for _, name := range DefaultConfigFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return LoadFile(path)
		}
	}
	return nil, nil // No config file found, not an error
}

// LoadFile loads configuration from a specific file.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Exists checks if a config file exists in the given directory.
func Exists(dir string) bool {
	for _, name := range DefaultConfigFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}
