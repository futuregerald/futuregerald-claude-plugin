package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetGlobals sets all package-level flags to their zero values.
func resetGlobals() {
	force = false
	dryRun = false
	nonInteract = false
	targetType = ""
	installMode = ""
}

// --- askInstallMode tests (no Target parameter) ---

func TestAskInstallMode_CLIFlag(t *testing.T) {
	tests := []struct {
		flag string
		want string
	}{
		{"full", modeFullInstall},
		{"config-only", modeConfigOnly},
		{"agents-only", modeAgentsOnly},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			installMode = tt.flag
			defer func() { installMode = "" }()

			reader := bufio.NewReader(strings.NewReader(""))
			got, err := askInstallMode(reader)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("askInstallMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAskInstallMode_NoTarget(t *testing.T) {
	// askInstallMode should not require a Target parameter
	// This test validates the new signature: func askInstallMode(reader *bufio.Reader) (string, error)
	reader := bufio.NewReader(strings.NewReader("1\n"))
	mode, err := askInstallMode(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != modeFullInstall {
		t.Errorf("mode = %q, want %q", mode, modeFullInstall)
	}
}

func TestAskInstallMode_InteractiveConfigOnly(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("2\n"))
	mode, err := askInstallMode(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != modeConfigOnly {
		t.Errorf("mode = %q, want %q", mode, modeConfigOnly)
	}
}

func TestAskInstallMode_InteractiveAgentsOnly(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("3\n"))
	mode, err := askInstallMode(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != modeAgentsOnly {
		t.Errorf("mode = %q, want %q", mode, modeAgentsOnly)
	}
}

// --- getTarget tests (mode-aware) ---

func TestGetTarget_ConfigOnlyFiltersEmptyConfigPath(t *testing.T) {
	// opencode and vscode have empty ConfigPath — should not appear in config-only mode
	reader := bufio.NewReader(strings.NewReader("1\n"))
	target, err := getTarget(reader, modeConfigOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default choice (1) should be claude, which has ConfigPath
	if target.ConfigPath == "" {
		t.Error("config-only mode returned a target with empty ConfigPath")
	}
}

func TestGetTarget_ConfigOnlyRejectsInvalidTarget(t *testing.T) {
	// -t opencode with config-only should error because opencode has no ConfigPath
	targetType = "opencode"
	defer func() { targetType = "" }()

	reader := bufio.NewReader(strings.NewReader(""))
	_, err := getTarget(reader, modeConfigOnly)
	if err == nil {
		t.Fatal("expected error for opencode with config-only mode, got nil")
	}
	if !strings.Contains(err.Error(), "not supported") && !strings.Contains(err.Error(), "config") {
		t.Errorf("error should mention config not supported, got: %v", err)
	}
}

func TestGetTarget_ConfigOnlyAcceptsValidTarget(t *testing.T) {
	targetType = "claude"
	defer func() { targetType = "" }()

	reader := bufio.NewReader(strings.NewReader(""))
	target, err := getTarget(reader, modeConfigOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != "Claude Code" {
		t.Errorf("target.Name = %q, want %q", target.Name, "Claude Code")
	}
}

func TestGetTarget_FullModeShowsAllTargets(t *testing.T) {
	// In full mode, all targets including opencode/vscode should be available
	targetType = "opencode"
	defer func() { targetType = "" }()

	reader := bufio.NewReader(strings.NewReader(""))
	target, err := getTarget(reader, modeFullInstall)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != "OpenCode" {
		t.Errorf("target.Name = %q, want %q", target.Name, "OpenCode")
	}
}

func TestGetTarget_AgentsOnlyAcceptsTarget(t *testing.T) {
	targetType = "claude"
	defer func() { targetType = "" }()

	reader := bufio.NewReader(strings.NewReader(""))
	target, err := getTarget(reader, modeAgentsOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.Name != "Claude Code" {
		t.Errorf("target.Name = %q, want %q", target.Name, "Claude Code")
	}
}

// --- generateConfigFile overwrite confirmation tests ---

func TestGenerateConfigFile_ExistingFilePromptsOverwrite(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(configPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	target := Target{
		Name:       "Claude Code",
		ConfigPath: "CLAUDE.md",
	}

	// Simulate user answering "n" (no overwrite)
	reader := bufio.NewReader(strings.NewReader("n\n"))

	// Save and restore globals
	origForce := force
	origDryRun := dryRun
	origNonInteract := nonInteract
	force = false
	dryRun = false
	nonInteract = false
	defer func() {
		force = origForce
		dryRun = origDryRun
		nonInteract = origNonInteract
	}()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	err := generateConfigFile(nil, target, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should still have original content (not overwritten)
	data, _ := os.ReadFile(configPath)
	if string(data) != "existing" {
		t.Errorf("file was overwritten despite user saying no")
	}
}

func TestGenerateConfigFile_ExistingFileForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(configPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	target := Target{
		Name:       "Claude Code",
		ConfigPath: "CLAUDE.md",
	}

	reader := bufio.NewReader(strings.NewReader(""))

	origForce := force
	origDryRun := dryRun
	force = true
	dryRun = false
	defer func() {
		force = origForce
		dryRun = origDryRun
	}()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	err := generateConfigFile(nil, target, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be overwritten
	data, _ := os.ReadFile(configPath)
	if string(data) == "existing" {
		t.Error("file was NOT overwritten despite --force")
	}
}

func TestGenerateConfigFile_ExistingFileNonInteractiveSkips(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(configPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	target := Target{
		Name:       "Claude Code",
		ConfigPath: "CLAUDE.md",
	}

	reader := bufio.NewReader(strings.NewReader(""))

	origForce := force
	origDryRun := dryRun
	origNonInteract := nonInteract
	force = false
	dryRun = false
	nonInteract = true
	defer func() {
		force = origForce
		dryRun = origDryRun
		nonInteract = origNonInteract
	}()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	err := generateConfigFile(nil, target, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should still have original content (skipped silently)
	data, _ := os.ReadFile(configPath)
	if string(data) != "existing" {
		t.Errorf("file was overwritten in non-interactive mode without --force")
	}
}

// --- askOverwriteAgents tests ---

func TestAskOverwriteAgents_NoExistingFiles(t *testing.T) {
	dir := t.TempDir()
	reader := bufio.NewReader(strings.NewReader(""))

	overwrite, err := askOverwriteAgents(reader, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No existing files → should proceed (return true)
	if !overwrite {
		t.Error("expected true when no existing agent files")
	}
}

func TestAskOverwriteAgents_ExistingFilesUserSaysNo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	origForce := force
	origNonInteract := nonInteract
	force = false
	nonInteract = false
	defer func() {
		force = origForce
		nonInteract = origNonInteract
	}()

	reader := bufio.NewReader(strings.NewReader("n\n"))

	overwrite, err := askOverwriteAgents(reader, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overwrite {
		t.Error("expected false when user says no")
	}
}

func TestAskOverwriteAgents_ExistingFilesUserSaysYes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	origForce := force
	origNonInteract := nonInteract
	force = false
	nonInteract = false
	defer func() {
		force = origForce
		nonInteract = origNonInteract
	}()

	reader := bufio.NewReader(strings.NewReader("y\n"))

	overwrite, err := askOverwriteAgents(reader, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overwrite {
		t.Error("expected true when user says yes")
	}
}

func TestAskOverwriteAgents_ForceBypassesPrompt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	origForce := force
	force = true
	defer func() { force = origForce }()

	reader := bufio.NewReader(strings.NewReader(""))

	overwrite, err := askOverwriteAgents(reader, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overwrite {
		t.Error("expected true with --force")
	}
}

func TestAskOverwriteAgents_NonInteractiveSkips(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	origForce := force
	origNonInteract := nonInteract
	force = false
	nonInteract = true
	defer func() {
		force = origForce
		nonInteract = origNonInteract
	}()

	reader := bufio.NewReader(strings.NewReader(""))

	overwrite, err := askOverwriteAgents(reader, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overwrite {
		t.Error("expected false in non-interactive mode without --force")
	}
}
