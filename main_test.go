package main

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// resetGlobals sets all package-level flags to their zero values.
func resetGlobals() {
	force = false
	dryRun = false
	nonInteract = false
	targetType = ""
	skipClaude = false
	tags = nil
	languages = nil
	fromSource = ""
	configFile = ""
	skipAgents = false
	skipCommands = false
	globalInstall = false
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

func TestAskOverwriteAgents_DanglingSymlinkPromptsOrForces(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	dir := t.TempDir()
	if err := os.Symlink("/nonexistent/agent.md", filepath.Join(dir, "plan-reviewer.md")); err != nil {
		t.Fatal(err)
	}

	nonInteract = true
	force = false
	reader := bufio.NewReader(strings.NewReader(""))
	overwrite, err := askOverwriteAgents(reader, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overwrite {
		t.Error("expected false in non-interactive mode without --force when dangling symlink exists")
	}

	force = true
	overwrite, err = askOverwriteAgents(reader, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overwrite {
		t.Error("expected true with --force when dangling symlink exists")
	}

	parent := t.TempDir()
	symlinkDir := filepath.Join(parent, "agents")
	if err := os.Symlink("/nonexistent/agents-dir", symlinkDir); err != nil {
		t.Fatal(err)
	}
	force = false
	nonInteract = true
	overwrite, err = askOverwriteAgents(reader, symlinkDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !overwrite {
		t.Error("expected true in non-interactive mode when agentsDest itself is a symlink so symlink is replaced")
	}

	nonInteract = false
	overwrite, err = askOverwriteAgents(bufio.NewReader(strings.NewReader("n\n")), symlinkDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overwrite {
		t.Error("expected false in interactive mode when user declines replacing symlinked agentsDest")
	}
}

func TestGetTarget_AntigravityAndGemini(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	for _, mode := range []string{modeFullInstall, modeConfigOnly, modeAgentsOnly} {
		for _, tc := range []struct {
			key  string
			name string
		}{
			{"antigravity", "Antigravity"},
			{"gemini", "Gemini CLI"},
		} {
			targetType = tc.key
			got, err := getTarget(bufio.NewReader(strings.NewReader("")), mode)
			if err != nil {
				t.Fatalf("getTarget(%q, %q) unexpected error: %v", tc.key, mode, err)
			}
			if got.Name != tc.name {
				t.Errorf("getTarget(%q, %q).Name = %q, want %q", tc.key, mode, got.Name, tc.name)
			}
			if got.ConfigPath != "GEMINI.md" {
				t.Errorf("getTarget(%q, %q).ConfigPath = %q, want GEMINI.md", tc.key, mode, got.ConfigPath)
			}
		}
	}

	targetType = ""
	nonInteract = false
	got1, err := getTarget(bufio.NewReader(strings.NewReader("1\n")), modeFullInstall)
	if err != nil || got1.Name != "Claude Code" {
		t.Fatalf("choice 1 = %v, %v; want Claude Code", got1.Name, err)
	}
	got2, err := getTarget(bufio.NewReader(strings.NewReader("2\n")), modeFullInstall)
	if err != nil || got2.Name != "Antigravity" {
		t.Fatalf("choice 2 = %v, %v; want Antigravity", got2.Name, err)
	}
	got3, err := getTarget(bufio.NewReader(strings.NewReader("3\n")), modeFullInstall)
	if err != nil || got3.Name != "Gemini CLI" {
		t.Fatalf("choice 3 = %v, %v; want Gemini CLI", got3.Name, err)
	}
}

func TestGenerateConfigFile_GeminiTargetsUseGeminiTemplate(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	for _, key := range []string{"antigravity", "gemini"} {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}

		force = true
		nonInteract = true
		target := targets[key]
		err := generateConfigFile(nil, target, bufio.NewReader(strings.NewReader("")))
		_ = os.Chdir(origDir)
		if err != nil {
			t.Fatalf("generateConfigFile(%s) error: %v", key, err)
		}

		data, err := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
		if err != nil {
			t.Fatalf("reading GEMINI.md for %s: %v", key, err)
		}
		text := string(data)
		if !strings.Contains(text, "/start") || !strings.Contains(text, "invoke_subagent") {
			t.Errorf("GEMINI.md for %s missing /start or invoke_subagent", key)
		}
		if strings.Contains(text, "CLAUDE.md") || strings.Contains(text, "TaskCreate") {
			t.Errorf("GEMINI.md for %s leaked CLAUDE.md or TaskCreate", key)
		}
	}
}

func TestEmbeddedContent_InstallsAntigravityAndGemini(t *testing.T) {
	resetGlobals()
	defer resetGlobals()

	for _, key := range []string{"antigravity", "gemini"} {
		projDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(projDir, ".gitignore"), []byte(".claude/*\n"), 0644); err != nil {
			t.Fatal(err)
		}
		origDir, _ := os.Getwd()
		if err := os.Chdir(projDir); err != nil {
			t.Fatal(err)
		}

		resetGlobals()
		targetType = key
		nonInteract = true
		force = true

		err := runInstall(nil, nil)
		_ = os.Chdir(origDir)
		if err != nil {
			t.Fatalf("runInstall(%s) project scope failed: %v", key, err)
		}

		if _, err := os.Stat(filepath.Join(projDir, "GEMINI.md")); err != nil {
			t.Errorf("%s project install missing GEMINI.md: %v", key, err)
		}
		if _, err := os.Stat(filepath.Join(projDir, "init-claude-md")); !os.IsNotExist(err) {
			t.Errorf("%s project install leaked commands into project root", key)
		}
		gitignoreData, _ := os.ReadFile(filepath.Join(projDir, ".gitignore"))
		if strings.Contains(string(gitignoreData), "!.claude/project.json") {
			t.Errorf("%s project install mutated .gitignore with .claude rule", key)
		}

		_, swarmErr := os.Stat(filepath.Join(projDir, ".agents", "skills", "peer-pipeline-swarm", "SKILL.md"))
		if key == "antigravity" && swarmErr != nil {
			t.Errorf("antigravity project install missing .agents/skills/peer-pipeline-swarm/SKILL.md: %v", swarmErr)
		}
		if key == "gemini" {
			if _, err := os.Stat(filepath.Join(projDir, ".gemini", "skills", "peer-pipeline-swarm", "SKILL.md")); !os.IsNotExist(err) {
				t.Errorf("gemini install should skip peer-pipeline-swarm/SKILL.md")
			}
		}

		expectedSkillRoot := ".agents"
		if key == "gemini" {
			expectedSkillRoot = ".gemini"
		}
		lifecycleData, err := os.ReadFile(filepath.Join(projDir, expectedSkillRoot, "skills", "lifecycle", "SKILL.md"))
		if err != nil {
			t.Fatalf("%s missing %s/skills/lifecycle/SKILL.md: %v", key, expectedSkillRoot, err)
		}
		if strings.Contains(string(lifecycleData), "TaskCreate") {
			t.Errorf("%s lifecycle/SKILL.md leaked TaskCreate", key)
		}
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	for _, key := range []string{"antigravity", "gemini"} {
		resetGlobals()
		targetType = key
		globalInstall = true
		nonInteract = true
		force = true
		if err := runInstall(nil, nil); err != nil {
			t.Fatalf("runInstall(%s, --global) failed: %v", key, err)
		}
	}
	if _, err := os.Stat(filepath.Join(tmpHome, ".gemini", "config", "skills", "peer-pipeline-swarm", "SKILL.md")); err != nil {
		t.Errorf("global antigravity missing ~/.gemini/config/skills/peer-pipeline-swarm/SKILL.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpHome, ".gemini", "skills", "lifecycle", "SKILL.md")); err != nil {
		t.Errorf("global gemini missing ~/.gemini/skills/lifecycle/SKILL.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpHome, ".gemini", "skills", "peer-pipeline-swarm", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("global gemini should not install peer-pipeline-swarm into ~/.gemini/skills")
	}
	agSkill, err := os.ReadFile(filepath.Join(tmpHome, ".gemini", "config", "skills", "subagent-driven-development", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agSkill), "~/.gemini/config/plugins/antigravity-subagents/agents/implementer.md") {
		t.Errorf("global antigravity skill was overwritten with gemini paths")
	}
	globalGeminiMD, err := os.ReadFile(filepath.Join(tmpHome, ".gemini", "GEMINI.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(globalGeminiMD), "~/.gemini/config/skills/antigravity-orchestration/SKILL.md") {
		t.Errorf("global ~/.gemini/GEMINI.md lost antigravity-orchestration path after gemini install")
	}

	piiRe := regexp.MustCompile(`geraldonyango|Gerald|[Cc]obalt|zombie|0d3ebe47|[Mm]olly|DL-[0-9X]|cobalthq|futuregerald|/Users/[a-ce-zA-Z]|/Users/d[^e]`)
	_ = fs.WalkDir(content, "antigravity", func(p string, d fs.DirEntry, wErr error) error {
		if wErr != nil || p == "antigravity/sync-from-local.sh" {
			return wErr
		}
		if d.IsDir() {
			if d.Name() == "__pycache__" || d.Name() == ".pytest_cache" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".pyc") {
			return nil
		}
		if strings.HasSuffix(p, ".local.md") || strings.HasSuffix(p, "/references/config.md") {
			t.Errorf("embedded antigravity contains forbidden local file: %s", p)
		}
		b, rErr := fs.ReadFile(content, p)
		if rErr != nil {
			t.Fatalf("reading embedded %s: %v", p, rErr)
		}
		if loc := piiRe.FindString(string(b)); loc != "" {
			t.Errorf("embedded %s matched forbidden PII/org pattern %q", p, loc)
		}
		return nil
	})

	symlinkProj := t.TempDir()
	secretTarget := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretTarget, []byte("DO_NOT_TOUCH"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secretTarget, filepath.Join(symlinkProj, "GEMINI.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secretTarget, filepath.Join(symlinkProj, ".gitignore")); err != nil {
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(symlinkProj); err != nil {
		t.Fatal(err)
	}
	resetGlobals()
	force = true
	nonInteract = true
	_ = generateConfigFile(nil, targets["antigravity"], bufio.NewReader(strings.NewReader("")))
	ensureGitignoreAllowsProjectJSON(bufio.NewReader(strings.NewReader("y\n")))
	_ = os.Chdir(origDir)
	secretData, _ := os.ReadFile(secretTarget)
	if string(secretData) != "DO_NOT_TOUCH" {
		t.Errorf("symlink target was overwritten by generateConfigFile or ensureGitignoreAllowsProjectJSON: %q", string(secretData))
	}
	resetGlobals()
	targetType = "antigravity"
	installMode = modeConfigOnly
	globalInstall = true
	nonInteract = true
	force = true
	if err := runInstall(nil, nil); err != nil {
		t.Fatalf("runInstall(antigravity, config-only, --global) failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpHome, ".gemini", "GEMINI.md")); err != nil {
		t.Errorf("global config-only missing ~/.gemini/GEMINI.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpHome, ".gemini", "config", "hooks.json")); err != nil {
		t.Errorf("global config-only missing ~/.gemini/config/hooks.json: %v", err)
	}

	resetGlobals()
	targetType = "cursor"
	installMode = modeConfigOnly
	globalInstall = true
	nonInteract = true
	if err := runInstall(nil, nil); err == nil {
		t.Errorf("expected error for --mode config-only --global with cursor target")
	}

	resetGlobals()
	targetType = "unknown"
	if err := runList(nil, nil); err == nil {
		t.Errorf("expected error from runList with unknown target")
	}

	resetGlobals()
	targetType = "antigravity"
	tags = []string{"go"}
	languages = []string{"go"}
	if err := runList(nil, nil); err != nil {
		t.Errorf("runList(antigravity) failed: %v", err)
	}
}
