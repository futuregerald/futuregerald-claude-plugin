package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Target       string   `yaml:"target"`
	Track        string   `yaml:"track"`
	Skills       []string `yaml:"skills"`
	Backup       *bool    `yaml:"backup"`
	Tags         []string `yaml:"tags"`
	Languages    []string `yaml:"languages"`
	SkipClaudeMD bool     `yaml:"skip_claude_md"`
	ConfigMode   string   `yaml:"config_mode"`
	ProjectDir   string   `yaml:"project_dir"`
	From         string   `yaml:"from"`
	Mode         string   `yaml:"mode"`
}

var DefaultConfigFiles = []string{
	".skill-installer.yaml",
	".skill-installer.yml",
	"skill-installer.yaml",
	"skill-installer.yml",
}

func Load(dir string) (*Config, error) {
	for _, name := range DefaultConfigFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return LoadFile(path)
		}
	}
	return nil, nil
}

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

func Exists(dir string) bool {
	for _, name := range DefaultConfigFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}
