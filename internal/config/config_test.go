package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndExists(t *testing.T) {
	dir := t.TempDir()
	if Exists(dir) {
		t.Fatal("expected Exists to be false in empty dir")
	}
	cfg, err := Load(dir)
	if err != nil || cfg != nil {
		t.Fatalf("expected nil, nil in empty dir; got %v, %v", cfg, err)
	}

	yamlText := "target: antigravity\ntrack: opinionated\nbackup: false\nskills:\n  - start\n  - writing-plans\n"
	cfgPath := filepath.Join(dir, ".skill-installer.yaml")
	if err := os.WriteFile(cfgPath, []byte(yamlText), 0644); err != nil {
		t.Fatal(err)
	}
	if !Exists(dir) {
		t.Fatal("expected Exists to be true")
	}
	loaded, err := Load(dir)
	if err != nil || loaded == nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.Target != "antigravity" || loaded.Track != "opinionated" {
		t.Errorf("unexpected target/track: %+v", loaded)
	}
	if loaded.Backup == nil || *loaded.Backup != false {
		t.Errorf("expected Backup=false, got %v", loaded.Backup)
	}
	if len(loaded.Skills) != 2 {
		t.Errorf("expected 2 skills, got %v", loaded.Skills)
	}
}
