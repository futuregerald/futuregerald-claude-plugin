package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePaths_24CellMatrix(t *testing.T) {
	winEnv := Env{
		GOOS:    "windows",
		GOARCH:  "amd64",
		HomeDir: `C:\Users\alice`,
		AppData: `C:\Users\alice\AppData\Roaming`,
	}
	macEnv := Env{
		GOOS:    "darwin",
		GOARCH:  "arm64",
		HomeDir: "/Users/alice",
	}
	linuxEnv := Env{
		GOOS:          "linux",
		GOARCH:        "amd64",
		HomeDir:       "/home/alice",
		XDGConfigHome: "/home/alice/.xdg",
	}

	tests := []struct {
		name       string
		env        Env
		harness    string
		scope      string
		proj       string
		wantSkills string
		wantAgents string
		wantCmds   string
		wantConfig string
		wantHooks  string
	}{
		{
			name:       "claude project darwin",
			env:        macEnv,
			harness:    "claude",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.claude/skills",
			wantAgents: "/work/app/.claude/agents",
			wantCmds:   "/work/app/.claude/commands",
			wantConfig: "/work/app/CLAUDE.md",
		},
		{
			name:       "claude project linux",
			env:        linuxEnv,
			harness:    "claude",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.claude/skills",
			wantAgents: "/work/app/.claude/agents",
			wantCmds:   "/work/app/.claude/commands",
			wantConfig: "/work/app/CLAUDE.md",
		},
		{
			name:       "claude project windows",
			env:        winEnv,
			harness:    "claude",
			scope:      "project",
			proj:       `C:\work\app`,
			wantSkills: `C:\work\app\.claude\skills`,
			wantAgents: `C:\work\app\.claude\agents`,
			wantCmds:   `C:\work\app\.claude\commands`,
			wantConfig: `C:\work\app\CLAUDE.md`,
		},
		{
			name:       "claude global darwin",
			env:        macEnv,
			harness:    "claude",
			scope:      "global",
			wantSkills: "/Users/alice/.claude/skills",
			wantAgents: "/Users/alice/.claude/agents",
			wantCmds:   "/Users/alice/.claude/commands",
			wantConfig: "/Users/alice/.claude/CLAUDE.md",
		},
		{
			name:       "claude global linux",
			env:        linuxEnv,
			harness:    "claude",
			scope:      "global",
			wantSkills: "/home/alice/.claude/skills",
			wantAgents: "/home/alice/.claude/agents",
			wantCmds:   "/home/alice/.claude/commands",
			wantConfig: "/home/alice/.claude/CLAUDE.md",
		},
		{
			name:       "claude global windows",
			env:        winEnv,
			harness:    "claude",
			scope:      "global",
			wantSkills: `C:\Users\alice\.claude\skills`,
			wantAgents: `C:\Users\alice\.claude\agents`,
			wantCmds:   `C:\Users\alice\.claude\commands`,
			wantConfig: `C:\Users\alice\.claude\CLAUDE.md`,
		},
		{
			name:       "antigravity project darwin",
			env:        macEnv,
			harness:    "antigravity",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.agents/skills",
			wantAgents: "/work/app/.agents/agents",
			wantConfig: "/work/app/GEMINI.md",
		},
		{
			name:       "antigravity project linux",
			env:        linuxEnv,
			harness:    "antigravity",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.agents/skills",
			wantAgents: "/work/app/.agents/agents",
			wantConfig: "/work/app/GEMINI.md",
		},
		{
			name:       "antigravity project windows",
			env:        winEnv,
			harness:    "antigravity",
			scope:      "project",
			proj:       `C:\work\app`,
			wantSkills: `C:\work\app\.agents\skills`,
			wantAgents: `C:\work\app\.agents\agents`,
			wantConfig: `C:\work\app\GEMINI.md`,
		},
		{
			name:       "antigravity global darwin",
			env:        macEnv,
			harness:    "antigravity",
			scope:      "global",
			wantSkills: "/Users/alice/.gemini/config/skills",
			wantAgents: "/Users/alice/.gemini/config/plugins/antigravity-subagents/agents",
			wantConfig: "/Users/alice/.gemini/GEMINI.md",
			wantHooks:  "/Users/alice/.gemini/config/hooks.json",
		},
		{
			name:       "antigravity global linux",
			env:        linuxEnv,
			harness:    "antigravity",
			scope:      "global",
			wantSkills: "/home/alice/.gemini/config/skills",
			wantAgents: "/home/alice/.gemini/config/plugins/antigravity-subagents/agents",
			wantConfig: "/home/alice/.gemini/GEMINI.md",
			wantHooks:  "/home/alice/.gemini/config/hooks.json",
		},
		{
			name:       "antigravity global windows",
			env:        winEnv,
			harness:    "antigravity",
			scope:      "global",
			wantSkills: `C:\Users\alice\.gemini\config\skills`,
			wantAgents: `C:\Users\alice\.gemini\config\plugins\antigravity-subagents\agents`,
			wantConfig: `C:\Users\alice\.gemini\GEMINI.md`,
			wantHooks:  `C:\Users\alice\.gemini\config\hooks.json`,
		},
		{
			name:       "zed project darwin",
			env:        macEnv,
			harness:    "zed",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.agents/skills",
			wantConfig: "/work/app/AGENTS.md",
		},
		{
			name:       "zed project linux",
			env:        linuxEnv,
			harness:    "zed",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.agents/skills",
			wantConfig: "/work/app/AGENTS.md",
		},
		{
			name:       "zed project windows",
			env:        winEnv,
			harness:    "zed",
			scope:      "project",
			proj:       `C:\work\app`,
			wantSkills: `C:\work\app\.agents\skills`,
			wantConfig: `C:\work\app\AGENTS.md`,
		},
		{
			name:       "zed global darwin",
			env:        macEnv,
			harness:    "zed",
			scope:      "global",
			wantSkills: "/Users/alice/.agents/skills",
			wantConfig: "/Users/alice/.config/zed/AGENTS.md",
		},
		{
			name:       "zed global linux xdg",
			env:        linuxEnv,
			harness:    "zed",
			scope:      "global",
			wantSkills: "/home/alice/.agents/skills",
			wantConfig: "/home/alice/.xdg/zed/AGENTS.md",
		},
		{
			name:       "zed global windows appdata",
			env:        winEnv,
			harness:    "zed",
			scope:      "global",
			wantSkills: `C:\Users\alice\.agents\skills`,
			wantConfig: `C:\Users\alice\AppData\Roaming\Zed\AGENTS.md`,
		},
		{
			name:       "opencode project darwin",
			env:        macEnv,
			harness:    "opencode",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.opencode/skills",
			wantAgents: "/work/app/.opencode/agents",
			wantCmds:   "/work/app/.opencode/commands",
			wantConfig: "/work/app/AGENTS.md",
		},
		{
			name:       "opencode project linux",
			env:        linuxEnv,
			harness:    "opencode",
			scope:      "project",
			proj:       "/work/app",
			wantSkills: "/work/app/.opencode/skills",
			wantAgents: "/work/app/.opencode/agents",
			wantCmds:   "/work/app/.opencode/commands",
			wantConfig: "/work/app/AGENTS.md",
		},
		{
			name:       "opencode project windows",
			env:        winEnv,
			harness:    "opencode",
			scope:      "project",
			proj:       `C:\work\app`,
			wantSkills: `C:\work\app\.opencode\skills`,
			wantAgents: `C:\work\app\.opencode\agents`,
			wantCmds:   `C:\work\app\.opencode\commands`,
			wantConfig: `C:\work\app\AGENTS.md`,
		},
		{
			name:       "opencode global darwin",
			env:        macEnv,
			harness:    "opencode",
			scope:      "global",
			wantSkills: "/Users/alice/.config/opencode/skills",
			wantAgents: "/Users/alice/.config/opencode/agents",
			wantCmds:   "/Users/alice/.config/opencode/commands",
			wantConfig: "/Users/alice/.config/opencode/AGENTS.md",
		},
		{
			name:       "opencode global linux xdg",
			env:        linuxEnv,
			harness:    "opencode",
			scope:      "global",
			wantSkills: "/home/alice/.xdg/opencode/skills",
			wantAgents: "/home/alice/.xdg/opencode/agents",
			wantCmds:   "/home/alice/.xdg/opencode/commands",
			wantConfig: "/home/alice/.xdg/opencode/AGENTS.md",
		},
		{
			name:       "opencode global windows",
			env:        winEnv,
			harness:    "opencode",
			scope:      "global",
			wantSkills: `C:\Users\alice\.config\opencode\skills`,
			wantAgents: `C:\Users\alice\.config\opencode\agents`,
			wantCmds:   `C:\Users\alice\.config\opencode\commands`,
			wantConfig: `C:\Users\alice\.config\opencode\AGENTS.md`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.env.ResolvePaths(tc.harness, tc.scope, tc.proj)
			if err != nil {
				t.Fatalf("ResolvePaths error: %v", err)
			}
			if got.SkillsDir != tc.wantSkills {
				t.Errorf("SkillsDir = %q, want %q", got.SkillsDir, tc.wantSkills)
			}
			if got.AgentsDir != tc.wantAgents {
				t.Errorf("AgentsDir = %q, want %q", got.AgentsDir, tc.wantAgents)
			}
			if got.CommandsDir != tc.wantCmds {
				t.Errorf("CommandsDir = %q, want %q", got.CommandsDir, tc.wantCmds)
			}
			if got.ConfigFile != tc.wantConfig {
				t.Errorf("ConfigFile = %q, want %q", got.ConfigFile, tc.wantConfig)
			}
			if tc.wantHooks != "" && got.HooksJSON != tc.wantHooks {
				t.Errorf("HooksJSON = %q, want %q", got.HooksJSON, tc.wantHooks)
			}
		})
	}
}

func TestAppStateDirAndRejectedHarnesses(t *testing.T) {
	winEnv := Env{GOOS: "windows", HomeDir: `C:\Users\alice`, AppData: `C:\Users\alice\AppData\Roaming`}
	if got := winEnv.AppStateDir(); got != `C:\Users\alice\AppData\Roaming\skill-installer` {
		t.Errorf("windows AppStateDir = %q", got)
	}

	linuxEnv := Env{GOOS: "linux", HomeDir: "/home/alice", XDGConfigHome: "/home/alice/.xdg"}
	if got := linuxEnv.AppStateDir(); got != "/home/alice/.xdg/skill-installer" {
		t.Errorf("linux AppStateDir = %q", got)
	}

	macEnv := Env{GOOS: "darwin", HomeDir: "/Users/alice"}
	if got := macEnv.AppStateDir(); got != "/Users/alice/.config/skill-installer" {
		t.Errorf("darwin AppStateDir = %q", got)
	}

	for _, bad := range []string{"copilot", "cursor", "vscode", "claude-desktop", "gemini"} {
		if _, err := macEnv.ResolvePaths(bad, "project", "."); err == nil {
			t.Errorf("expected error for retired harness %q", bad)
		}
	}
}

func TestWindowsPythonHookCommandAndSweep(t *testing.T) {
	cmd := WindowsPythonHookCommand(`C:\Users\alice\.gemini`, func(name string) (string, error) {
		if name == "py" {
			return `C:\Windows\py.exe`, nil
		}
		return "", errors.New("not found")
	})
	want := `"C:/Windows/py.exe" -3 "C:/Users/alice/.gemini/config/hooks/autonomous_breaker.py"`
	if cmd != want {
		t.Errorf("WindowsPythonHookCommand = %q, want %q", cmd, want)
	}

	sanitized := WindowsPythonHookCommand(`C:\Users\alice"%$foo\.gemini`, func(name string) (string, error) {
		return "", errors.New("not found")
	})
	if strings.ContainsAny(sanitized, "%$`") {
		t.Errorf("expected unsafe shell chars stripped from WindowsPythonHookCommand, got %q", sanitized)
	}

	tmp := t.TempDir()
	exe := filepath.Join(tmp, "skill-installer.exe")
	oldExe := exe + ".old"
	if err := os.WriteFile(oldExe, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	SweepStaleExecutable(exe)
	if _, err := os.Stat(oldExe); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed by SweepStaleExecutable", oldExe)
	}
}
