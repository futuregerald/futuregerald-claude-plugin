//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func sandboxEnv(sandboxHome, xdgConfig string) []string {
	appData := filepath.Join(sandboxHome, "AppData", "Roaming")
	breakerState := filepath.Join(sandboxHome, ".gemini", "state")
	breakerLog := filepath.Join(sandboxHome, ".gemini", "metrics", "breaker.jsonl")
	var filtered []string
	blockedPrefixes := []string{
		"HOME=",
		"USERPROFILE=",
		"XDG_CONFIG_HOME=",
		"APPDATA=",
		"BREAKER_STATE_DIR=",
		"BREAKER_LOG_PATH=",
		"BREAKER_DISABLED=",
		"BREAKER_PASSTHROUGH_OUT=",
		"BREAKER_MODE=",
		"BREAKER_RULE_",
		"BATON_",
		"NO_COLOR=",
		"CI=",
	}
	for _, kv := range os.Environ() {
		skip := false
		for _, prefix := range blockedPrefixes {
			if strings.HasPrefix(kv, prefix) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, kv)
		}
	}
	filtered = append(filtered,
		"HOME="+sandboxHome,
		"USERPROFILE="+sandboxHome,
		"XDG_CONFIG_HOME="+xdgConfig,
		"APPDATA="+appData,
		"BREAKER_STATE_DIR="+breakerState,
		"BREAKER_LOG_PATH="+breakerLog,
		"NO_COLOR=1",
	)
	return filtered
}

func assertIsolatedSandboxHome(t *testing.T, sandboxHome string) {
	t.Helper()
	if sandboxHome == "" {
		t.Fatal("sandboxHome must not be empty")
	}
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		cleanReal := filepath.Clean(u.HomeDir)
		cleanSandbox := filepath.Clean(sandboxHome)
		if cleanReal == cleanSandbox {
			t.Fatalf("sandboxHome %q must never equal real user home %q", cleanSandbox, cleanReal)
		}
	}
}

func repoRootDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if filepath.Base(wd) == "e2e" {
		return filepath.Dir(wd)
	}
	return wd
}

func buildOrLocateSkillInstaller(t *testing.T) string {
	t.Helper()
	if envBin := os.Getenv("SKILL_INSTALLER_BIN"); envBin != "" {
		if info, err := os.Stat(envBin); err == nil && !info.IsDir() {
			return envBin
		}
	}
	if os.Getenv("SKILL_INSTALLER_E2E_CONTAINER") == "1" {
		if resolved, err := exec.LookPath("skill-installer"); err == nil {
			return resolved
		}
	}
	root := repoRootDir(t)
	binDir := t.TempDir()
	binName := "skill-installer"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(binDir, binName)
	goBin := "go"
	if goroot := runtime.GOROOT(); goroot != "" {
		goExe := "go"
		if runtime.GOOS == "windows" {
			goExe = "go.exe"
		}
		cand := filepath.Join(goroot, "bin", goExe)
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			goBin = cand
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, goBin, "build", "-ldflags", "-X main.version=4.0.0", "-o", binPath, ".")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building skill-installer failed: %v\n%s", err, string(out))
	}
	return binPath
}

func runCLIInSandbox(t *testing.T, binPath, workDir, sandboxHome, xdgConfig string, args ...string) (string, error) {
	t.Helper()
	assertIsolatedSandboxHome(t, sandboxHome)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, args...)
	if workDir != "" {
		cmd.Dir = workDir
	} else {
		cmd.Dir = sandboxHome
	}
	cmd.Env = sandboxEnv(sandboxHome, xdgConfig)
	cmd.Stdin = bytes.NewReader(nil)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func mustRunCLIInSandbox(t *testing.T, binPath, workDir, sandboxHome, xdgConfig string, args ...string) string {
	t.Helper()
	out, err := runCLIInSandbox(t, binPath, workDir, sandboxHome, xdgConfig, args...)
	if err != nil {
		t.Fatalf("skill-installer %v failed: %v\noutput:\n%s", args, err, out)
	}
	return out
}

func runExternalBinary(t *testing.T, name, sandboxHome, xdgConfig string, args ...string) (string, error) {
	t.Helper()
	assertIsolatedSandboxHome(t, sandboxHome)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = sandboxHome
	cmd.Env = sandboxEnv(sandboxHome, xdgConfig)
	cmd.Stdin = bytes.NewReader(nil)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func mustExistFile(t *testing.T, path string) []byte {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("expected regular file at %s, found directory", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return data
}

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to not exist, stat err=%v", path, err)
	}
}

func assertNoFrontmatterModelOrEffort(t *testing.T, filePath string, data []byte) {
	t.Helper()
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return
	}
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			break
		}
		if strings.HasPrefix(trimmed, "model:") || strings.HasPrefix(trimmed, "effort:") {
			t.Fatalf("%s leaked frontmatter %q in non-Claude harness", filePath, trimmed)
		}
	}
}

func assertNoRawTemplatePlaceholders(t *testing.T, filePath string, data []byte) {
	t.Helper()
	text := string(data)
	for _, token := range []string{"<!-- LANGUAGE_SPECIFIC -->", "{{PROJECT_NAME}}", "{{HARNESS}}"} {
		if strings.Contains(text, token) {
			t.Fatalf("%s contains unexpanded placeholder %q", filePath, token)
		}
	}
}

func TestContainerHarnessesE2E(t *testing.T) {
	if os.Getenv("SKILL_INSTALLER_E2E_CONTAINER") != "1" {
		t.Skip("skipping container-only E2E test outside container (set SKILL_INSTALLER_E2E_CONTAINER=1 or run make test-e2e)")
	}
	binPath := buildOrLocateSkillInstaller(t)
	runVerifyAllHarnessesInSandbox(t, binPath, true)
}

func TestHostDesktopAppsCompatibilitySandbox(t *testing.T) {
	binPath := buildOrLocateSkillInstaller(t)
	if runtime.GOOS == "darwin" {
		for _, appBundle := range []string{
			"/Applications/Antigravity.app",
			"/Applications/Claude.app",
			"/Applications/Zed.app",
		} {
			if info, err := os.Stat(appBundle); err == nil && info.IsDir() {
				plistPath := filepath.Join(appBundle, "Contents", "Info.plist")
				if _, pErr := os.Stat(plistPath); pErr != nil {
					t.Errorf("installed desktop bundle %s missing Info.plist: %v", appBundle, pErr)
				}
			}
		}
	}
	runVerifyAllHarnessesInSandbox(t, binPath, false)
}

func runVerifyAllHarnessesInSandbox(t *testing.T, binPath string, requireRealCLIs bool) {
	t.Helper()
	sandboxHome := t.TempDir()
	xdgConfig := filepath.Join(sandboxHome, ".xdg-config")
	assertIsolatedSandboxHome(t, sandboxHome)

	verOut := mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig, "version")
	if !strings.Contains(verOut, "4.0.0") {
		t.Fatalf("expected version output to contain 4.0.0, got %q", verOut)
	}

	listOut := mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig, "list", "--long")
	for _, requiredSkill := range []string{
		"start",
		"test-driven-development",
		"comprehensive-code-review",
		"plugin-updater",
	} {
		if !strings.Contains(listOut, requiredSkill) {
			t.Fatalf("skill-installer list --long missing %q", requiredSkill)
		}
	}
	if strings.Contains(listOut, "baton-pass") {
		t.Fatal("default claude skill-installer list --long must not include antigravity-only baton-pass")
	}

	agListOut := mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig, "list", "-t", "antigravity", "--long")
	if !strings.Contains(agListOut, "baton-pass") {
		t.Fatal("skill-installer list -t antigravity --long missing baton-pass")
	}

	guideOut := mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig, "guide", "-t", "antigravity", "--track", "opinionated", "-y")
	if !strings.Contains(guideOut, "THE BASICS: SLASH COMMANDS vs. SKILLS") || !strings.Contains(guideOut, "WHAT YOU CAN TYPE NOW") {
		t.Fatalf("skill-installer guide output missing expected sections:\n%s", guideOut)
	}

	initDir := t.TempDir()
	initPath := filepath.Join(initDir, ".skill-installer.yaml")
	mustRunCLIInSandbox(t, binPath, initDir, sandboxHome, xdgConfig, "init", "-o", initPath)
	initBytes := mustExistFile(t, initPath)
	if !strings.Contains(string(initBytes), "target:") {
		t.Fatalf("expected generated config to contain target:, got:\n%s", string(initBytes))
	}

	verifyClaudeDesktopAndCLI(t, binPath, sandboxHome, xdgConfig, requireRealCLIs)
	verifyAntigravityDesktopAndLiveBreakerHook(t, binPath, sandboxHome, xdgConfig)
	verifyZedDesktop(t, binPath, sandboxHome, xdgConfig)
	verifyOpenCodeCLI(t, binPath, sandboxHome, xdgConfig, requireRealCLIs)
	verifyProjectLanguageDetectionAndMergeModes(t, binPath, sandboxHome, xdgConfig)
}

func verifyClaudeDesktopAndCLI(t *testing.T, binPath, sandboxHome, xdgConfig string, requireRealCLI bool) {
	t.Helper()
	assertIsolatedSandboxHome(t, sandboxHome)

	if _, err := exec.LookPath("claude"); err == nil {
		out, cErr := runExternalBinary(t, "claude", sandboxHome, xdgConfig, "--version")
		if cErr != nil {
			t.Fatalf("claude --version failed: %v\n%s", cErr, out)
		}
		if strings.TrimSpace(out) == "" {
			t.Fatal("claude --version returned empty output")
		}
	} else if requireRealCLI {
		t.Fatal("claude CLI binary not found on PATH inside container")
	}

	claudeRoot := filepath.Join(sandboxHome, ".claude")
	macDesktopDir := filepath.Join(sandboxHome, "Library", "Application Support", "Claude")
	linuxDesktopDir := filepath.Join(xdgConfig, "Claude")
	if err := os.MkdirAll(claudeRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(macDesktopDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(linuxDesktopDir, 0755); err != nil {
		t.Fatal(err)
	}

	desktopMCPJSON := `{"mcpServers":{"filesystem":{"command":"npx","args":["-y","@modelcontextprotocol/server-filesystem","/tmp"]}},"globalShortcut":"CommandOrControl+Shift+Space"}`
	claudeSettingsJSON := `{"theme":"dark","permissions":{"allow":["Read","Grep"]}}`
	customGlobalClaudeMD := "# Personal Claude Desktop & Code Preferences\n\nAlways answer concisely.\n"

	macDesktopCfg := filepath.Join(macDesktopDir, "claude_desktop_config.json")
	linuxDesktopCfg := filepath.Join(linuxDesktopDir, "claude_desktop_config.json")
	settingsCfg := filepath.Join(claudeRoot, "settings.json")
	globalClaudeMD := filepath.Join(claudeRoot, "CLAUDE.md")

	if err := os.WriteFile(macDesktopCfg, []byte(desktopMCPJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(linuxDesktopCfg, []byte(desktopMCPJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsCfg, []byte(claudeSettingsJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalClaudeMD, []byte(customGlobalClaudeMD), 0644); err != nil {
		t.Fatal(err)
	}

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"-t", "claude", "-g", "--track", "opinionated", "--merge-config", "--backup", "-y")

	mergedClaudeMD := string(mustExistFile(t, globalClaudeMD))
	if !strings.Contains(mergedClaudeMD, "# Personal Claude Desktop & Code Preferences") {
		t.Fatal("merged global CLAUDE.md lost user header")
	}
	if !strings.Contains(mergedClaudeMD, "<!-- BEGIN SKILL-INSTALLER WORKFLOW RULES -->") {
		t.Fatal("merged global CLAUDE.md missing managed workflow block")
	}
	if strings.Contains(mergedClaudeMD, "baton-pass") {
		t.Fatal("Claude global CLAUDE.md must never reference Antigravity-only baton-pass")
	}

	for _, skill := range []string{"start", "begin", "lifecycle", "test-driven-development", "comprehensive-code-review", "plugin-updater"} {
		mustExistFile(t, filepath.Join(claudeRoot, "skills", skill, "SKILL.md"))
	}
	mustNotExist(t, filepath.Join(claudeRoot, "skills", "baton-pass", "SKILL.md"))
	mustNotExist(t, filepath.Join(claudeRoot, "skills", "antigravity-orchestration", "SKILL.md"))
	mustNotExist(t, filepath.Join(claudeRoot, "skills", "peer-pipeline-swarm", "SKILL.md"))

	for _, agent := range []string{"implementer.md", "code-quality-reviewer.md", "security-reviewer.md", "adversarial-plan-reviewer.md"} {
		agentBytes := mustExistFile(t, filepath.Join(claudeRoot, "agents", agent))
		if !strings.Contains(string(agentBytes), "model:") {
			t.Fatalf("expected Claude agent %s to retain model: frontmatter", agent)
		}
	}
	for _, cmdFile := range []string{"start.md", "shape.md", "plugin-update.md"} {
		mustExistFile(t, filepath.Join(claudeRoot, "commands", cmdFile))
	}

	if string(mustExistFile(t, macDesktopCfg)) != desktopMCPJSON {
		t.Fatal("Claude Desktop macOS config was unexpectedly modified")
	}
	if string(mustExistFile(t, linuxDesktopCfg)) != desktopMCPJSON {
		t.Fatal("Claude Desktop Linux config was unexpectedly modified")
	}
	if string(mustExistFile(t, settingsCfg)) != claudeSettingsJSON {
		t.Fatal("Claude settings.json was unexpectedly modified")
	}

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"-t", "claude", "-g", "--track", "product", "--merge-config", "--backup", "--force", "-y")
	mustExistFile(t, filepath.Join(claudeRoot, "skills", "write-a-prd", "SKILL.md"))
	mustExistFile(t, filepath.Join(claudeRoot, "skills", "craftsmans-wisdom", "SKILL.md"))
	mustNotExist(t, filepath.Join(claudeRoot, "agents", "implementer.md"))

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"uninstall", "-t", "claude", "-g", "-y")

	afterUninstallClaudeMD := string(mustExistFile(t, globalClaudeMD))
	if !strings.Contains(afterUninstallClaudeMD, "# Personal Claude Desktop & Code Preferences") {
		t.Fatal("uninstall removed user's custom rules from global CLAUDE.md")
	}
	if strings.Contains(afterUninstallClaudeMD, "<!-- BEGIN SKILL-INSTALLER WORKFLOW RULES -->") {
		t.Fatal("uninstall failed to strip managed workflow block from global CLAUDE.md")
	}
	mustNotExist(t, filepath.Join(claudeRoot, "skills", "write-a-prd", "SKILL.md"))
	if string(mustExistFile(t, macDesktopCfg)) != desktopMCPJSON {
		t.Fatal("Claude Desktop config modified during uninstall")
	}
	if string(mustExistFile(t, settingsCfg)) != claudeSettingsJSON {
		t.Fatal("Claude settings.json modified during uninstall")
	}
}

func verifyAntigravityDesktopAndLiveBreakerHook(t *testing.T, binPath, sandboxHome, xdgConfig string) {
	t.Helper()
	assertIsolatedSandboxHome(t, sandboxHome)

	if _, err := exec.LookPath("antigravity"); err == nil {
		out, aErr := runExternalBinary(t, "antigravity", sandboxHome, xdgConfig, "--version")
		if aErr != nil {
			t.Fatalf("antigravity --version failed: %v\n%s", aErr, out)
		}
	}

	geminiRoot := filepath.Join(sandboxHome, ".gemini")
	geminiConfigDir := filepath.Join(geminiRoot, "config")
	projectsDir := filepath.Join(geminiConfigDir, "projects")
	rulesDir := filepath.Join(geminiConfigDir, "rules")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}

	configJSON := `{"model":"gemini-4-argon-medium","theme":"dark"}`
	mcpConfigJSON := `{"mcpServers":{"prism":{"command":"prism-mcp"}}}`
	skillsJSON := `{"enabled":true}`
	customOrgLocalRule := "# Local Org Rule\n\nKeep this untouched.\n"
	coTenantHooksJSON := `{
  "custom-audit-logger": {
    "enabled": true,
    "PreToolUse": [
      {
        "matcher": "run_command",
        "hooks": [
          {
            "type": "command",
            "command": "echo audit",
            "timeout": 2
          }
        ]
      }
    ]
  }
}`

	if err := os.WriteFile(filepath.Join(geminiConfigDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(geminiConfigDir, "mcp_config.json"), []byte(mcpConfigJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(geminiConfigDir, "skills.json"), []byte(skillsJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "org.local.md"), []byte(customOrgLocalRule), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(geminiConfigDir, "hooks.json"), []byte(coTenantHooksJSON), 0644); err != nil {
		t.Fatal(err)
	}

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"-t", "antigravity", "-g", "--track", "opinionated", "--no-backup", "--force", "-y")

	geminiMDBytes := mustExistFile(t, filepath.Join(geminiRoot, "GEMINI.md"))
	assertNoRawTemplatePlaceholders(t, "GEMINI.md", geminiMDBytes)
	if !strings.Contains(string(geminiMDBytes), "baton-pass") {
		t.Fatal("expected Antigravity global GEMINI.md to reference baton-pass")
	}
	mustExistFile(t, filepath.Join(rulesDir, "org.example.md"))
	if string(mustExistFile(t, filepath.Join(rulesDir, "org.local.md"))) != customOrgLocalRule {
		t.Fatal("org.local.md was modified during Antigravity install")
	}

	mustExistFile(t, filepath.Join(geminiConfigDir, "plugins", "antigravity-subagents", "plugin.json"))
	for _, agent := range []string{"implementer.md", "code-quality-reviewer.md", "security-reviewer.md", "plan-reviewer.md"} {
		agentPath := filepath.Join(geminiConfigDir, "plugins", "antigravity-subagents", "agents", agent)
		agentData := mustExistFile(t, agentPath)
		assertNoFrontmatterModelOrEffort(t, agentPath, agentData)
	}

	for _, agExclusive := range []string{"baton-pass", "antigravity-orchestration", "peer-pipeline-swarm"} {
		skillPath := filepath.Join(geminiConfigDir, "skills", agExclusive, "SKILL.md")
		data := mustExistFile(t, skillPath)
		assertNoFrontmatterModelOrEffort(t, skillPath, data)
	}

	hooksJSONPath := filepath.Join(geminiConfigDir, "hooks.json")
	hooksBytes := mustExistFile(t, hooksJSONPath)
	var parsedHooks map[string]any
	if err := json.Unmarshal(hooksBytes, &parsedHooks); err != nil {
		t.Fatalf("unmarshaling installed hooks.json: %v", err)
	}
	if _, ok := parsedHooks["custom-audit-logger"]; !ok {
		t.Fatal("expected co-tenant custom-audit-logger hook preserved in hooks.json")
	}
	breakerEntry, ok := parsedHooks["autonomous-circuit-breaker"].(map[string]any)
	if !ok {
		t.Fatal("missing autonomous-circuit-breaker in hooks.json")
	}
	preToolUse, ok := breakerEntry["PreToolUse"].([]any)
	if !ok || len(preToolUse) != 21 {
		t.Fatalf("expected 21 PreToolUse matchers in hooks.json, got %d", len(preToolUse))
	}

	hookShPath := filepath.Join(geminiConfigDir, "hooks", "autonomous-breaker.sh")
	mustExistFile(t, hookShPath)
	mustExistFile(t, filepath.Join(geminiConfigDir, "hooks", "autonomous_breaker.py"))
	mustExistFile(t, filepath.Join(geminiConfigDir, "hooks", "breaker_rules.py"))

	executeLiveBreakerScenarios(t, hookShPath, sandboxHome, xdgConfig)

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"-t", "antigravity", "-g", "--track", "copy-polish", "--backup", "--force", "-y")
	mustNotExist(t, filepath.Join(geminiConfigDir, "hooks", "autonomous-breaker.sh"))
	mustNotExist(t, filepath.Join(geminiConfigDir, "plugins", "antigravity-subagents", "plugin.json"))
	mustExistFile(t, filepath.Join(geminiConfigDir, "skills", "copy-edit", "SKILL.md"))

	afterSwitchHooksBytes := mustExistFile(t, hooksJSONPath)
	var afterSwitchHooks map[string]any
	if err := json.Unmarshal(afterSwitchHooksBytes, &afterSwitchHooks); err != nil {
		t.Fatalf("unmarshaling hooks.json after track switch: %v", err)
	}
	if _, hasBreaker := afterSwitchHooks["autonomous-circuit-breaker"]; hasBreaker {
		t.Fatal("expected autonomous-circuit-breaker removed when switching to copy-polish track")
	}
	if _, hasCoTenant := afterSwitchHooks["custom-audit-logger"]; !hasCoTenant {
		t.Fatal("expected custom-audit-logger preserved when switching to copy-polish track")
	}

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"restore", "-t", "antigravity", "-g", "-y")
	mustExistFile(t, filepath.Join(geminiConfigDir, "hooks", "autonomous-breaker.sh"))
	mustExistFile(t, filepath.Join(geminiConfigDir, "plugins", "antigravity-subagents", "plugin.json"))

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"uninstall", "-t", "antigravity", "-g", "--restore-config", "-y")
	mustNotExist(t, filepath.Join(geminiRoot, "GEMINI.md"))
	mustNotExist(t, filepath.Join(geminiConfigDir, "hooks", "autonomous-breaker.sh"))
	mustNotExist(t, filepath.Join(geminiConfigDir, "plugins", "antigravity-subagents", "plugin.json"))
	if string(mustExistFile(t, filepath.Join(geminiConfigDir, "config.json"))) != configJSON {
		t.Fatal("Antigravity config.json modified after uninstall")
	}
	if string(mustExistFile(t, filepath.Join(geminiConfigDir, "mcp_config.json"))) != mcpConfigJSON {
		t.Fatal("Antigravity mcp_config.json modified after uninstall")
	}
	if string(mustExistFile(t, filepath.Join(geminiConfigDir, "skills.json"))) != skillsJSON {
		t.Fatal("Antigravity skills.json modified after uninstall")
	}
	if string(mustExistFile(t, filepath.Join(rulesDir, "org.local.md"))) != customOrgLocalRule {
		t.Fatal("Antigravity org.local.md modified after uninstall")
	}
}

func executeLiveBreakerScenarios(t *testing.T, hookShPath, sandboxHome, xdgConfig string) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found on PATH; skipping live hook execution")
	}

	writeTranscript := func(convID string, inputTokens int) string {
		tDir := filepath.Join(sandboxHome, ".gemini", "antigravity", "brain", convID, ".system_generated", "logs")
		if err := os.MkdirAll(tDir, 0755); err != nil {
			t.Fatal(err)
		}
		tFile := filepath.Join(tDir, "transcript.jsonl")
		lines := fmt.Sprintf(
			"{\"type\":\"USER_INPUT\",\"content\":\"check status\"}\n{\"type\":\"PLANNER_RESPONSE\",\"input_tokens\":%d,\"output_tokens\":0}\n",
			inputTokens,
		)
		if err := os.WriteFile(tFile, []byte(lines), 0644); err != nil {
			t.Fatal(err)
		}
		return tFile
	}

	invokeHook := func(payload map[string]any) string {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "bash", hookShPath)
		cmd.Dir = sandboxHome
		cmd.Env = sandboxEnv(sandboxHome, xdgConfig)
		cmd.Stdin = bytes.NewReader(raw)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("autonomous-breaker.sh exited with error: %v\noutput: %s", err, string(out))
		}
		return string(out)
	}

	tLow := writeTranscript("conv-low", 20000)
	outLow := invokeHook(map[string]any{
		"conversationId": "conv-low",
		"transcriptPath": tLow,
		"toolCall": map[string]any{
			"name": "read_file",
			"args": map[string]any{"Path": "/tmp/sample.txt"},
		},
	})
	if outLow != "" {
		t.Fatalf("expected empty stdout for low-token transcript, got %q", outLow)
	}

	tWarn := writeTranscript("conv-warn", 210000)
	outWarn1 := invokeHook(map[string]any{
		"conversationId": "conv-warn",
		"transcriptPath": tWarn,
		"toolCall": map[string]any{
			"name": "read_file",
			"args": map[string]any{"Path": filepath.Join(sandboxHome, "main.go")},
		},
	})
	var warnDecision map[string]string
	if err := json.Unmarshal([]byte(outWarn1), &warnDecision); err != nil {
		t.Fatalf("expected JSON deny on first warn call, got %q (err=%v)", outWarn1, err)
	}
	if warnDecision["decision"] != "deny" || !strings.Contains(warnDecision["reason"], "CIRCUIT BREAKER (Baton Warn): Context estimate 210000 >= 205000") {
		t.Fatalf("unexpected warn decision: %+v", warnDecision)
	}
	stateWarnFile := filepath.Join(sandboxHome, ".gemini", "state", "baton-conv-warn.json")
	mustExistFile(t, stateWarnFile)

	outWarn2 := invokeHook(map[string]any{
		"conversationId": "conv-warn",
		"transcriptPath": tWarn,
		"toolCall": map[string]any{
			"name": "read_file",
			"args": map[string]any{"Path": filepath.Join(sandboxHome, "main.go")},
		},
	})
	if outWarn2 != "" {
		t.Fatalf("expected second call at warn threshold to pass through with empty stdout, got %q", outWarn2)
	}

	tBlock := writeTranscript("conv-block", 245000)
	outBlock := invokeHook(map[string]any{
		"conversationId": "conv-block",
		"transcriptPath": tBlock,
		"toolCall": map[string]any{
			"name": "run_command",
			"args": map[string]any{"CommandLine": "pytest"},
		},
	})
	var blockDecision map[string]string
	if err := json.Unmarshal([]byte(outBlock), &blockDecision); err != nil {
		t.Fatalf("expected JSON deny on block call, got %q (err=%v)", outBlock, err)
	}
	if blockDecision["decision"] != "deny" || !strings.Contains(blockDecision["reason"], "CIRCUIT BREAKER (Baton Block): Context estimate 245000 >= 235000") {
		t.Fatalf("unexpected block decision: %+v", blockDecision)
	}

	handoffPath := filepath.Join(sandboxHome, ".gemini", "handoffs", "proj", "2026-11-14-120000-slug-ACTIVE_CONTEXT.md")
	outUnlockWrite := invokeHook(map[string]any{
		"conversationId": "conv-block",
		"transcriptPath": tBlock,
		"toolCall": map[string]any{
			"name": "write_file",
			"args": map[string]any{"Path": handoffPath},
		},
	})
	if outUnlockWrite != "" {
		t.Fatalf("expected ACTIVE_CONTEXT.md write to pass through at block threshold, got %q", outUnlockWrite)
	}

	outAfterUnlock := invokeHook(map[string]any{
		"conversationId": "conv-block",
		"transcriptPath": tBlock,
		"toolCall": map[string]any{
			"name": "run_command",
			"args": map[string]any{"CommandLine": "git status --porcelain"},
		},
	})
	if outAfterUnlock != "" {
		t.Fatalf("expected run_command within +50k handoff unlock window to pass through, got %q", outAfterUnlock)
	}
}

func verifyZedDesktop(t *testing.T, binPath, sandboxHome, xdgConfig string) {
	t.Helper()
	assertIsolatedSandboxHome(t, sandboxHome)

	if _, err := exec.LookPath("zed"); err == nil {
		_, _ = runExternalBinary(t, "zed", sandboxHome, xdgConfig, "--version")
	}

	zedConfigDir := filepath.Join(xdgConfig, "zed")
	if err := os.MkdirAll(zedConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsJSON := `{"vim_mode":true,"buffer_font_size":14}`
	keymapJSON := `[{"context":"Workspace","bindings":{"cmd-s":"workspace::Save"}}]`
	if err := os.WriteFile(filepath.Join(zedConfigDir, "settings.json"), []byte(settingsJSON), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zedConfigDir, "keymap.json"), []byte(keymapJSON), 0644); err != nil {
		t.Fatal(err)
	}

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"-t", "zed", "-g", "--track", "opinionated", "--backup", "--force", "-y")

	zedAgentsMD := mustExistFile(t, filepath.Join(zedConfigDir, "AGENTS.md"))
	assertNoRawTemplatePlaceholders(t, "zed/AGENTS.md", zedAgentsMD)
	if strings.Contains(string(zedAgentsMD), "baton-pass") {
		t.Fatal("Zed AGENTS.md must not reference Antigravity-only baton-pass")
	}

	zedSkillsDir := filepath.Join(sandboxHome, ".agents", "skills")
	for _, skill := range []string{"start", "test-driven-development", "comprehensive-code-review", "plugin-updater"} {
		sPath := filepath.Join(zedSkillsDir, skill, "SKILL.md")
		sBytes := mustExistFile(t, sPath)
		assertNoFrontmatterModelOrEffort(t, sPath, sBytes)
	}
	mustNotExist(t, filepath.Join(zedSkillsDir, "baton-pass", "SKILL.md"))

	if _, err := runCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"-t", "zed", "-g", "--mode", "agents-only", "-y"); err == nil {
		t.Fatal("expected zed with --mode agents-only to fail")
	}

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"uninstall", "-t", "zed", "-g", "--restore-config", "-y")
	mustNotExist(t, filepath.Join(zedConfigDir, "AGENTS.md"))
	mustNotExist(t, filepath.Join(zedSkillsDir, "start", "SKILL.md"))
	if string(mustExistFile(t, filepath.Join(zedConfigDir, "settings.json"))) != settingsJSON {
		t.Fatal("Zed settings.json was modified during install/uninstall")
	}
	if string(mustExistFile(t, filepath.Join(zedConfigDir, "keymap.json"))) != keymapJSON {
		t.Fatal("Zed keymap.json was modified during install/uninstall")
	}
}

func verifyOpenCodeCLI(t *testing.T, binPath, sandboxHome, xdgConfig string, requireRealCLI bool) {
	t.Helper()
	assertIsolatedSandboxHome(t, sandboxHome)

	if _, err := exec.LookPath("opencode"); err == nil {
		out, oErr := runExternalBinary(t, "opencode", sandboxHome, xdgConfig, "--version")
		if oErr != nil {
			t.Fatalf("opencode --version failed: %v\n%s", oErr, out)
		}
		if strings.TrimSpace(out) == "" {
			t.Fatal("opencode --version returned empty output")
		}
	} else if requireRealCLI {
		t.Fatal("opencode CLI binary not found on PATH inside container")
	}

	ocRoot := filepath.Join(xdgConfig, "opencode")
	for idx, track := range []string{"opinionated", "basics", "product", "copy-polish"} {
		backupArg := "--no-backup"
		if idx == 0 {
			backupArg = "--backup"
		}
		mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
			"-t", "opencode", "-g", "--track", track, backupArg, "--force", "-y")
		agentsMD := mustExistFile(t, filepath.Join(ocRoot, "AGENTS.md"))
		assertNoRawTemplatePlaceholders(t, "opencode/AGENTS.md", agentsMD)
		if strings.Contains(string(agentsMD), "baton-pass") {
			t.Fatalf("OpenCode AGENTS.md leaked baton-pass on track %s", track)
		}
		mustExistFile(t, filepath.Join(ocRoot, "skills", "plugin-updater", "SKILL.md"))
		mustNotExist(t, filepath.Join(ocRoot, "skills", "baton-pass", "SKILL.md"))
		if track == "opinionated" {
			for _, ag := range []string{"implementer.md", "code-quality-reviewer.md", "security-reviewer.md"} {
				p := filepath.Join(ocRoot, "agents", ag)
				b := mustExistFile(t, p)
				assertNoFrontmatterModelOrEffort(t, p, b)
			}
			mustExistFile(t, filepath.Join(ocRoot, "commands", "start.md"))
		}
	}

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"-t", "opencode", "-g", "--track", "custom", "--skills", "test-driven-development,copywriting", "--no-backup", "--force", "-y")
	mustExistFile(t, filepath.Join(ocRoot, "skills", "test-driven-development", "SKILL.md"))
	mustExistFile(t, filepath.Join(ocRoot, "skills", "copywriting", "SKILL.md"))

	mustRunCLIInSandbox(t, binPath, sandboxHome, sandboxHome, xdgConfig,
		"uninstall", "-t", "opencode", "-g", "--restore-config", "-y")
	mustNotExist(t, filepath.Join(ocRoot, "AGENTS.md"))
	mustNotExist(t, filepath.Join(ocRoot, "skills", "test-driven-development", "SKILL.md"))
}

func verifyProjectLanguageDetectionAndMergeModes(t *testing.T, binPath, sandboxHome, xdgConfig string) {
	t.Helper()
	assertIsolatedSandboxHome(t, sandboxHome)

	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{"name":"demo","dependencies":{"react":"^19.0.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "pyproject.toml"), []byte("[project]\nname = \"demo\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, harness := range []string{"claude", "antigravity", "zed", "opencode"} {
		mustRunCLIInSandbox(t, binPath, projDir, sandboxHome, xdgConfig,
			"-t", harness, "--project-dir", projDir, "--track", "basics", "--backup", "--force", "-y")
	}

	claudeMD := mustExistFile(t, filepath.Join(projDir, "CLAUDE.md"))
	geminiMD := mustExistFile(t, filepath.Join(projDir, "GEMINI.md"))
	agentsMD := mustExistFile(t, filepath.Join(projDir, "AGENTS.md"))
	assertNoRawTemplatePlaceholders(t, "proj/CLAUDE.md", claudeMD)
	assertNoRawTemplatePlaceholders(t, "proj/GEMINI.md", geminiMD)
	assertNoRawTemplatePlaceholders(t, "proj/AGENTS.md", agentsMD)

	customContent := "# Project Custom Header\n\nKeep me intact.\n"
	if err := os.WriteFile(filepath.Join(projDir, "CLAUDE.md"), []byte(customContent), 0644); err != nil {
		t.Fatal(err)
	}

	mustRunCLIInSandbox(t, binPath, projDir, sandboxHome, xdgConfig,
		"-t", "claude", "--project-dir", projDir, "--track", "copy-polish", "--config-mode", "skip", "-y")
	if string(mustExistFile(t, filepath.Join(projDir, "CLAUDE.md"))) != customContent {
		t.Fatal("expected --config-mode skip to leave CLAUDE.md untouched")
	}
	mustExistFile(t, filepath.Join(projDir, ".claude", "skills", "copy-edit", "SKILL.md"))

	mustRunCLIInSandbox(t, binPath, projDir, sandboxHome, xdgConfig,
		"-t", "claude", "--project-dir", projDir, "--track", "opinionated", "--config-mode", "merge", "-y")
	merged := string(mustExistFile(t, filepath.Join(projDir, "CLAUDE.md")))
	if !strings.Contains(merged, "# Project Custom Header") || !strings.Contains(merged, "<!-- BEGIN SKILL-INSTALLER WORKFLOW RULES -->") {
		t.Fatal("expected --config-mode merge to preserve header and append managed workflow rules")
	}

	mustRunCLIInSandbox(t, binPath, projDir, sandboxHome, xdgConfig,
		"-t", "claude", "--project-dir", projDir, "--track", "basics", "--config-mode", "replace", "--force", "-y")
	replaced := string(mustExistFile(t, filepath.Join(projDir, "CLAUDE.md")))
	if strings.Contains(replaced, "# Project Custom Header") {
		t.Fatal("expected --config-mode replace to overwrite custom header")
	}
}
