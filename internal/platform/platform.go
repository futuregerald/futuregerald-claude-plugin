package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
)

type Env struct {
	GOOS          string
	GOARCH        string
	HomeDir       string
	AppData       string
	XDGConfigHome string
}

type HarnessPaths struct {
	Harness     string
	Scope       string
	RootDir     string
	SkillsDir   string
	AgentsDir   string
	CommandsDir string
	ConfigFile  string
	RulesDir    string
	HooksDir    string
	HooksJSON   string
	PluginJSON  string
	GeminiHome  string
}

func EnvFromOS() Env {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	home := os.Getenv("HOME")
	if home == "" && goos == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}

	appData := os.Getenv("APPDATA")
	if appData == "" && goos == "windows" && home != "" {
		appData = joinForOS(goos, home, "AppData", "Roaming")
	}

	return Env{
		GOOS:          goos,
		GOARCH:        goarch,
		HomeDir:       home,
		AppData:       appData,
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
	}
}

func hasWindowsDriveOrBackslash(parts []string) bool {
	for _, p := range parts {
		if strings.Contains(p, `\`) || (len(p) >= 2 && p[1] == ':') {
			return true
		}
	}
	return false
}

func joinForOS(goos string, parts ...string) string {
	sep := "/"
	if goos == "windows" && (runtime.GOOS == "windows" || hasWindowsDriveOrBackslash(parts)) {
		sep = `\`
	}
	leadingRoot := false
	for _, p := range parts {
		if p == "" {
			continue
		}
		if sep == "/" && strings.HasPrefix(p, "/") {
			leadingRoot = true
		}
		break
	}
	var cleaned []string
	for i, p := range parts {
		if p == "" {
			continue
		}
		norm := strings.ReplaceAll(p, `\`, sep)
		norm = strings.ReplaceAll(norm, "/", sep)
		if i > 0 {
			norm = strings.TrimLeft(norm, sep)
		}
		if len(cleaned) > 0 || i < len(parts)-1 {
			norm = strings.TrimRight(norm, sep)
		}
		if norm != "" {
			cleaned = append(cleaned, norm)
		}
	}
	joined := strings.Join(cleaned, sep)
	if leadingRoot && !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}

func (e Env) AppStateDir() string {
	if e.GOOS == "windows" {
		base := e.AppData
		if base == "" {
			base = joinForOS(e.GOOS, e.HomeDir, "AppData", "Roaming")
		}
		return joinForOS(e.GOOS, base, "skill-installer")
	}
	if e.XDGConfigHome != "" {
		return joinForOS(e.GOOS, e.XDGConfigHome, "skill-installer")
	}
	return joinForOS(e.GOOS, e.HomeDir, ".config", "skill-installer")
}

func (e Env) StateFilePath() string {
	return joinForOS(e.GOOS, e.AppStateDir(), "state.json")
}

func (e Env) BackupsDir() string {
	return joinForOS(e.GOOS, e.AppStateDir(), "backups")
}

func (e Env) ResolvePaths(harness, scope, projectDir string) (HarnessPaths, error) {
	if !catalog.IsSupportedHarness(harness) {
		return HarnessPaths{}, fmt.Errorf("unsupported harness %q (supported: claude, antigravity, zed, opencode)", harness)
	}
	if scope == "" {
		scope = "project"
	}
	if scope != "project" && scope != "global" {
		return HarnessPaths{}, fmt.Errorf("unsupported scope %q (expected project or global)", scope)
	}

	if scope == "project" {
		root := projectDir
		if root == "" {
			root = "."
		}
		switch harness {
		case catalog.HarnessClaude:
			return HarnessPaths{
				Harness:     harness,
				Scope:       scope,
				RootDir:     root,
				SkillsDir:   joinForOS(e.GOOS, root, ".claude", "skills"),
				AgentsDir:   joinForOS(e.GOOS, root, ".claude", "agents"),
				CommandsDir: joinForOS(e.GOOS, root, ".claude", "commands"),
				ConfigFile:  joinForOS(e.GOOS, root, "CLAUDE.md"),
			}, nil
		case catalog.HarnessAntigravity:
			return HarnessPaths{
				Harness:    harness,
				Scope:      scope,
				RootDir:    root,
				SkillsDir:  joinForOS(e.GOOS, root, ".agents", "skills"),
				AgentsDir:  joinForOS(e.GOOS, root, ".agents", "agents"),
				ConfigFile: joinForOS(e.GOOS, root, "GEMINI.md"),
			}, nil
		case catalog.HarnessZed:
			return HarnessPaths{
				Harness:    harness,
				Scope:      scope,
				RootDir:    root,
				SkillsDir:  joinForOS(e.GOOS, root, ".agents", "skills"),
				ConfigFile: joinForOS(e.GOOS, root, "AGENTS.md"),
			}, nil
		case catalog.HarnessOpenCode:
			return HarnessPaths{
				Harness:     harness,
				Scope:       scope,
				RootDir:     root,
				SkillsDir:   joinForOS(e.GOOS, root, ".opencode", "skills"),
				AgentsDir:   joinForOS(e.GOOS, root, ".opencode", "agents"),
				CommandsDir: joinForOS(e.GOOS, root, ".opencode", "commands"),
				ConfigFile:  joinForOS(e.GOOS, root, "AGENTS.md"),
			}, nil
		}
	}

	switch harness {
	case catalog.HarnessClaude:
		root := joinForOS(e.GOOS, e.HomeDir, ".claude")
		return HarnessPaths{
			Harness:     harness,
			Scope:       scope,
			RootDir:     root,
			SkillsDir:   joinForOS(e.GOOS, root, "skills"),
			AgentsDir:   joinForOS(e.GOOS, root, "agents"),
			CommandsDir: joinForOS(e.GOOS, root, "commands"),
			ConfigFile:  joinForOS(e.GOOS, root, "CLAUDE.md"),
		}, nil
	case catalog.HarnessAntigravity:
		geminiHome := joinForOS(e.GOOS, e.HomeDir, ".gemini")
		return HarnessPaths{
			Harness:    harness,
			Scope:      scope,
			RootDir:    geminiHome,
			GeminiHome: geminiHome,
			SkillsDir:  joinForOS(e.GOOS, geminiHome, "config", "skills"),
			AgentsDir:  joinForOS(e.GOOS, geminiHome, "config", "plugins", "antigravity-subagents", "agents"),
			PluginJSON: joinForOS(e.GOOS, geminiHome, "config", "plugins", "antigravity-subagents", "plugin.json"),
			ConfigFile: joinForOS(e.GOOS, geminiHome, "GEMINI.md"),
			RulesDir:   joinForOS(e.GOOS, geminiHome, "config", "rules"),
			HooksDir:   joinForOS(e.GOOS, geminiHome, "config", "hooks"),
			HooksJSON:  joinForOS(e.GOOS, geminiHome, "config", "hooks.json"),
		}, nil
	case catalog.HarnessZed:
		skillsDir := joinForOS(e.GOOS, e.HomeDir, ".agents", "skills")
		var zedRoot string
		if e.GOOS == "windows" {
			appData := e.AppData
			if appData == "" {
				appData = joinForOS(e.GOOS, e.HomeDir, "AppData", "Roaming")
			}
			zedRoot = joinForOS(e.GOOS, appData, "Zed")
		} else {
			cfgBase := e.XDGConfigHome
			if cfgBase == "" {
				cfgBase = joinForOS(e.GOOS, e.HomeDir, ".config")
			}
			zedRoot = joinForOS(e.GOOS, cfgBase, "zed")
		}
		return HarnessPaths{
			Harness:    harness,
			Scope:      scope,
			RootDir:    zedRoot,
			SkillsDir:  skillsDir,
			ConfigFile: joinForOS(e.GOOS, zedRoot, "AGENTS.md"),
		}, nil
	case catalog.HarnessOpenCode:
		cfgBase := e.XDGConfigHome
		if cfgBase == "" {
			cfgBase = joinForOS(e.GOOS, e.HomeDir, ".config")
		}
		ocRoot := joinForOS(e.GOOS, cfgBase, "opencode")
		return HarnessPaths{
			Harness:     harness,
			Scope:       scope,
			RootDir:     ocRoot,
			SkillsDir:   joinForOS(e.GOOS, ocRoot, "skills"),
			AgentsDir:   joinForOS(e.GOOS, ocRoot, "agents"),
			CommandsDir: joinForOS(e.GOOS, ocRoot, "commands"),
			ConfigFile:  joinForOS(e.GOOS, ocRoot, "AGENTS.md"),
		}, nil
	}

	return HarnessPaths{}, fmt.Errorf("unhandled harness %q", harness)
}

func hasUnsafeShellChars(s string) bool {
	return strings.ContainsAny(s, "\"%$`&|;<>\r\n\x00")
}

func sanitizeShellPath(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if strings.IndexByte("\"%$`&|;<>\r\n\x00", c) >= 0 {
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func isAbsForOS(p string) bool {
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) {
		return true
	}
	return len(p) >= 3 && ((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) && p[1] == ':' && (p[2] == '/' || p[2] == '\\')
}

func WindowsPythonHookCommand(geminiHome string, lookPath func(string) (string, error)) string {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	launcher := "python"
	for _, candidate := range []string{"python", "py", "python3"} {
		if resolved, err := lookPath(candidate); err == nil {
			cleanResolved := sanitizeShellPath(strings.TrimSpace(resolved))
			slashResolved := strings.ReplaceAll(filepath.ToSlash(cleanResolved), `\`, "/")
			if isAbsForOS(slashResolved) {
				if candidate == "py" {
					launcher = fmt.Sprintf("%q -3", slashResolved)
				} else {
					launcher = fmt.Sprintf("%q", slashResolved)
				}
			} else if candidate == "py" {
				launcher = "py -3"
			} else {
				launcher = candidate
			}
			break
		}
	}
	cleanHome := sanitizeShellPath(geminiHome)
	slashHome := strings.ReplaceAll(filepath.ToSlash(cleanHome), `\`, "/")
	slashHome = strings.TrimRight(slashHome, "/")
	scriptPath := slashHome + "/config/hooks/autonomous_breaker.py"
	return fmt.Sprintf("%s %q", launcher, scriptPath)
}

func SweepStaleExecutable(exePath string) {
	if exePath == "" {
		return
	}
	_ = os.Remove(exePath + ".old")
}
