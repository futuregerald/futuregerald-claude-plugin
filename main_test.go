package main

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func resetGlobals() {
	force = false
	dryRun = false
	nonInteract = false
	targetType = ""
	trackName = ""
	selectedSkills = nil
	backupFlag = true
	noBackupFlag = false
	backupExplicit = false
	skipClaude = false
	tags = nil
	languages = nil
	fromSource = ""
	configFile = ""
	skipAgents = false
	skipCommands = false
	globalInstall = false
	projectInstall = false
	installMode = ""
	listLong = false
	listCategory = ""
	updateCheck = false
	updateApply = false
	updateSelf = false
	updateJSON = false
	restoreID = ""
	restoreFile = ""
	restoreConfig = false
	initOutput = ""
	updateRunner = nil
}

type mockGHRunner struct {
	out map[string][]byte
}

func (m mockGHRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	k := name + " " + strings.Join(args, " ")
	return m.out[k], nil
}

func TestAskInstallMode_AndGetTarget4Harnesses(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	for _, tc := range []struct {
		flag string
		want string
	}{
		{"full", modeFullInstall},
		{"config-only", modeConfigOnly},
		{"agents-only", modeAgentsOnly},
	} {
		installMode = tc.flag
		got, err := askInstallMode(bufio.NewReader(strings.NewReader("")))
		if err != nil || got != tc.want {
			t.Fatalf("askInstallMode(%q) = %q, %v", tc.flag, got, err)
		}
	}
	installMode = ""

	for _, key := range []string{"claude", "antigravity", "zed", "opencode"} {
		targetType = key
		got, err := getTarget(bufio.NewReader(strings.NewReader("")), modeFullInstall)
		if err != nil || got.Harness != key {
			t.Fatalf("getTarget(%q) = %+v, %v", key, got, err)
		}
	}

	targetType = "zed"
	if _, err := getTarget(bufio.NewReader(strings.NewReader("")), modeAgentsOnly); err == nil {
		t.Errorf("expected error for zed in agents-only mode")
	}

	for _, retired := range []string{"copilot", "cursor", "vscode", "claude-desktop", "gemini"} {
		targetType = retired
		if _, err := getTarget(bufio.NewReader(strings.NewReader("")), modeFullInstall); err == nil {
			t.Errorf("expected error for retired target %q", retired)
		}
	}
}

func TestGenerateConfigFile_AndOverwrite(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	configPath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(configPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := generateConfigFile(nil, targets["claude"], bufio.NewReader(strings.NewReader("n\n"))); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(configPath)
	if string(data) != "existing" {
		t.Errorf("expected existing CLAUDE.md preserved when user answered n")
	}

	force = true
	if err := generateConfigFile(nil, targets["claude"], bufio.NewReader(strings.NewReader(""))); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(configPath)
	if string(data) == "existing" {
		t.Errorf("expected CLAUDE.md overwritten with --force")
	}
}

func TestEndToEnd_4HarnessesBackupRestoreUninstallGuideAndUpdate(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".xdg"))
	t.Setenv("APPDATA", filepath.Join(tmpHome, "AppData", "Roaming"))

	for _, h := range []string{"claude", "antigravity", "zed", "opencode"} {
		projDir := t.TempDir()
		origDir, _ := os.Getwd()
		if err := os.Chdir(projDir); err != nil {
			t.Fatal(err)
		}

		resetGlobals()
		targetType = h
		trackName = "basics"
		nonInteract = true
		force = true
		backupFlag = true

		err := runInstall(nil, nil)
		_ = os.Chdir(origDir)
		if err != nil {
			t.Fatalf("runInstall(%s, project) failed: %v", h, err)
		}

		cfgName := targets[h].ConfigPath
		if _, err := os.Stat(filepath.Join(projDir, cfgName)); err != nil {
			t.Errorf("%s project install missing %s: %v", h, cfgName, err)
		}
		skillsDir := filepath.Join(projDir, filepath.FromSlash(targets[h].SkillsPath))
		if _, err := os.Stat(filepath.Join(skillsDir, "plugin-updater", "SKILL.md")); err != nil {
			t.Errorf("%s project install missing plugin-updater skill: %v", h, err)
		}
		if h != "antigravity" {
			if _, err := os.Stat(filepath.Join(skillsDir, "baton-pass", "SKILL.md")); !os.IsNotExist(err) {
				t.Errorf("%s install must never include antigravity-only baton-pass", h)
			}
		}
	}

	for _, h := range []string{"claude", "antigravity", "zed", "opencode"} {
		resetGlobals()
		targetType = h
		if h == "claude" || h == "antigravity" {
			trackName = "opinionated"
		} else {
			trackName = "custom"
			selectedSkills = []string{"test-driven-development", "ui-design"}
		}
		globalInstall = true
		nonInteract = true
		force = true
		backupFlag = true
		if err := runInstall(nil, nil); err != nil {
			t.Fatalf("runInstall(%s, --global) failed: %v", h, err)
		}
	}

	if _, err := os.Stat(filepath.Join(tmpHome, ".gemini", "config", "skills", "baton-pass", "SKILL.md")); err != nil {
		t.Errorf("global antigravity missing baton-pass/SKILL.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpHome, ".gemini", "config", "hooks.json")); err != nil {
		t.Errorf("global antigravity opinionated missing hooks.json: %v", err)
	}

	resetGlobals()
	targetType = "zed"
	trackName = "product"
	nonInteract = true
	if err := runGuide(nil, nil); err != nil {
		t.Errorf("runGuide failed: %v", err)
	}

	resetGlobals()
	targetType = "antigravity"
	trackName = "copy-polish"
	listLong = true
	if err := runList(nil, nil); err != nil {
		t.Errorf("runList --long failed: %v", err)
	}

	resetGlobals()
	targetType = "antigravity"
	globalInstall = true
	updateCheck = true
	updateJSON = true
	updateRunner = mockGHRunner{
		out: map[string][]byte{
			"gh api repos/futuregerald/futuregerald-claude-plugin/commits/main":    []byte(`{"sha":"abc999","commit":{"message":"feat: new skill"}}`),
			"gh api repos/futuregerald/futuregerald-claude-plugin/releases/latest": []byte(`{"tag_name":"v4.0.0"}`),
		},
	}
	if err := runUpdate(nil, nil); err != nil {
		t.Errorf("runUpdate --check --json failed: %v", err)
	}

	resetGlobals()
	targetType = "antigravity"
	nonInteract = true
	if err := runRestore(nil, nil); err != nil {
		t.Errorf("runRestore failed: %v", err)
	}

	resetGlobals()
	targetType = "claude"
	globalInstall = true
	nonInteract = true
	if err := runUninstall(nil, nil); err != nil {
		t.Errorf("runUninstall failed: %v", err)
	}

	globalPIIRe := regexp.MustCompile(`geraldonyango|zombie|0d3ebe47|cobalthq|/Users/geraldonyango`)
	antigravityPIIRe := regexp.MustCompile(`geraldonyango|Gerald|[Cc]obalt|zombie|0d3ebe47|[Mm]olly|DL-[0-9X]|cobalthq|futuregerald|/Users/[a-ce-zA-Z]|/Users/d[^e]`)
	for _, root := range []string{"shared", "claude", "antigravity", "zed", "opencode"} {
		_ = fs.WalkDir(content, root, func(p string, d fs.DirEntry, wErr error) error {
			if wErr != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(p, "LICENSE.txt") {
				return nil
			}
			b, rErr := fs.ReadFile(content, p)
			if rErr != nil {
				t.Fatalf("reading %s: %v", p, rErr)
			}
			sanitized := strings.NewReplacer("zombie UI", "", "zombie buildings", "").Replace(string(b))
			if loc := globalPIIRe.FindString(sanitized); loc != "" {
				t.Errorf("embedded %s matched forbidden PII pattern %q", p, loc)
			}
			if root == "antigravity" {
				if loc := antigravityPIIRe.FindString(sanitized); loc != "" {
					t.Errorf("embedded %s matched forbidden antigravity PII pattern %q", p, loc)
				}
			}
			return nil
		})
	}
}

func TestUntrustedFromSourceAndRerunStatePreservation(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".xdg"))

	projDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(projDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := os.WriteFile(filepath.Join(projDir, ".skill-installer.yaml"), []byte("from: http://127.0.0.1:9999/evil.tar.gz\n"), 0644); err != nil {
		t.Fatal(err)
	}

	targetType = "claude"
	trackName = "basics"
	nonInteract = true
	force = true
	if err := runInstall(nil, nil); err != nil {
		t.Fatalf("runInstall should ignore auto-discovered CWD from field, got error: %v", err)
	}
	if fromSource != "" {
		t.Fatalf("expected fromSource to remain empty when -c is not explicitly set, got %q", fromSource)
	}

	resetGlobals()
	targetType = "claude"
	nonInteract = true
	fromSource = "http://example.com/archive.tar.gz"
	if err := runInstall(nil, nil); err == nil {
		t.Fatalf("expected non-https fromSource URL to be rejected")
	}

	resetGlobals()
	targetType = "claude"
	trackName = "basics"
	nonInteract = true
	force = false
	if err := runInstall(nil, nil); err != nil {
		t.Fatalf("rerun runInstall without --force failed: %v", err)
	}

	resetGlobals()
	targetType = "claude"
	projectInstall = true
	nonInteract = true
	if err := runUninstall(nil, nil); err != nil {
		t.Fatalf("runUninstall after non-force rerun failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projDir, ".claude", "skills", "test-driven-development", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected uninstall after non-force rerun to remove tracked skills, err=%v", err)
	}
}
