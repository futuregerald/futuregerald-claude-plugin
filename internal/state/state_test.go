package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/backup"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
)

func TestStoreSaveFindAndUninstallPreservesUserAssetsAndHooks(t *testing.T) {
	tmp := t.TempDir()
	env := platform.Env{
		GOOS:    "darwin",
		GOARCH:  "arm64",
		HomeDir: tmp,
	}

	cat, err := catalog.Load(catalog.FS)
	if err != nil {
		t.Fatalf("catalog.Load error: %v", err)
	}

	paths, err := env.ResolvePaths("antigravity", "global", "")
	if err != nil {
		t.Fatal(err)
	}

	pluginSkillFile := filepath.Join(paths.SkillsDir, "start", "SKILL.md")
	localNoteFile := filepath.Join(paths.SkillsDir, "start", "references", "team.local.md")
	userSkillFile := filepath.Join(paths.SkillsDir, "my-custom-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(localNoteFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(userSkillFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pluginSkillFile, []byte("# start"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localNoteFile, []byte("keep local"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userSkillFile, []byte("# user skill"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(paths.HooksDir, 0755); err != nil {
		t.Fatal(err)
	}
	breakerPy := filepath.Join(paths.HooksDir, "autonomous_breaker.py")
	if err := os.WriteFile(breakerPy, []byte("print('ok')"), 0644); err != nil {
		t.Fatal(err)
	}
	coTenantHooks := `{"impeccable-design-detector":{"enabled":true},"autonomous-circuit-breaker":{"enabled":true}}`
	if err := os.WriteFile(paths.HooksJSON, []byte(coTenantHooks), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(env.StateFilePath())
	legacy, ok := DiscoverLegacyReceipt(env, cat, "antigravity", "global", "")
	if !ok || legacy == nil {
		t.Fatalf("expected DiscoverLegacyReceipt to detect v3.x install")
	}
	if legacy.Track != "opinionated" || !legacy.HooksMerged {
		t.Errorf("unexpected legacy receipt: %+v", legacy)
	}

	if err := store.Upsert(*legacy); err != nil {
		t.Fatalf("store.Upsert error: %v", err)
	}
	found, exists, err := store.Find("antigravity", "global", paths.RootDir)
	if err != nil || !exists || found == nil {
		t.Fatalf("store.Find failed: %v, %v", exists, err)
	}

	res, err := Uninstall(env, cat, *found, backup.NewManager(env.BackupsDir()), false)
	if err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}
	if !res.HooksCleaned {
		t.Errorf("expected HooksCleaned=true")
	}
	if _, err := os.Stat(pluginSkillFile); !os.IsNotExist(err) {
		t.Errorf("expected pluginSkillFile removed")
	}
	if _, err := os.Stat(localNoteFile); err != nil {
		t.Errorf("expected localNoteFile (*.local.md) preserved: %v", err)
	}
	if _, err := os.Stat(userSkillFile); err != nil {
		t.Errorf("expected user-authored my-custom-skill preserved: %v", err)
	}
	if _, err := os.Stat(breakerPy); !os.IsNotExist(err) {
		t.Errorf("expected autonomous_breaker.py removed")
	}
	hooksAfter, err := os.ReadFile(paths.HooksJSON)
	if err != nil {
		t.Fatalf("expected hooks.json preserved for co-tenant hook: %v", err)
	}
	if !strings.Contains(string(hooksAfter), "impeccable-design-detector") || strings.Contains(string(hooksAfter), "autonomous-circuit-breaker") {
		t.Errorf("unexpected hooks.json after uninstall:\n%s", string(hooksAfter))
	}

	symRepo := filepath.Join(t.TempDir(), "external-skills")
	if err := os.MkdirAll(filepath.Join(symRepo, "start"), 0755); err != nil {
		t.Fatal(err)
	}
	externalSkill := filepath.Join(symRepo, "start", "SKILL.md")
	if err := os.WriteFile(externalSkill, []byte("# external"), 0644); err != nil {
		t.Fatal(err)
	}
	symProj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(symProj, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(symRepo, filepath.Join(symProj, ".claude", "skills")); err != nil {
		t.Fatal(err)
	}
	if _, ok := DiscoverLegacyReceipt(env, cat, "claude", "project", symProj); ok {
		t.Errorf("DiscoverLegacyReceipt must not follow symlinked .claude/skills directory")
	}
}

func TestDiscoverLegacyReceiptSweepsRetiredV3SkillsAndCommands(t *testing.T) {
	tmp := t.TempDir()
	env := platform.Env{
		GOOS:    "darwin",
		GOARCH:  "arm64",
		HomeDir: tmp,
	}

	v4FS := fstest.MapFS{
		"tracks.yaml": &fstest.MapFile{
			Data: []byte("version: \"4.0.0\"\ntracks:\n  - id: opinionated\n    name: Opinionated\n    skills: [start]\nskills:\n  - name: start\n    category: workflow\n    slash_command: /start\n    short: Start\n    long: Start workflow\n    starter_prompt: /start\n"),
		},
		"claude/commands/start.md": &fstest.MapFile{
			Data: []byte("# /start\n"),
		},
	}
	cat, err := catalog.Load(v4FS)
	if err != nil {
		t.Fatalf("catalog.Load(v4FS) error: %v", err)
	}

	projDir := filepath.Join(tmp, "legacy-proj")
	paths, err := env.ResolvePaths("claude", "project", projDir)
	if err != nil {
		t.Fatal(err)
	}

	activeSkillFile := filepath.Join(paths.SkillsDir, "start", "SKILL.md")
	retiredToTicketsFile := filepath.Join(paths.SkillsDir, "to-tickets", "SKILL.md")
	retiredImplementFile := filepath.Join(paths.SkillsDir, "implement", "SKILL.md")
	retiredLocalNote := filepath.Join(paths.SkillsDir, "implement", "references", "team.local.md")
	retiredCmdFile := filepath.Join(paths.CommandsDir, "to-tickets.md")
	userCustomCmdFile := filepath.Join(paths.CommandsDir, "my-custom.md")

	for _, f := range []string{activeSkillFile, retiredToTicketsFile, retiredImplementFile, retiredLocalNote, retiredCmdFile, userCustomCmdFile} {
		if err := os.MkdirAll(filepath.Dir(f), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("legacy content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	legacy, ok := DiscoverLegacyReceipt(env, cat, "claude", "project", projDir)
	if !ok || legacy == nil {
		t.Fatalf("expected DiscoverLegacyReceipt to detect legacy install")
	}

	for _, s := range legacy.Skills {
		if s == "to-tickets" || s == "implement" {
			t.Errorf("retired skill %q must not appear in Receipt.Skills: %v", s, legacy.Skills)
		}
	}
	if len(legacy.Skills) != 1 || legacy.Skills[0] != "start" {
		t.Errorf("expected Receipt.Skills=[start], got %v", legacy.Skills)
	}

	installedSet := make(map[string]bool, len(legacy.InstalledFiles))
	for _, f := range legacy.InstalledFiles {
		installedSet[filepath.Clean(f)] = true
	}
	for _, wantTracked := range []string{activeSkillFile, retiredToTicketsFile, retiredImplementFile, retiredCmdFile} {
		if !installedSet[filepath.Clean(wantTracked)] {
			t.Errorf("expected %s in InstalledFiles, got %v", wantTracked, legacy.InstalledFiles)
		}
	}
	if installedSet[filepath.Clean(retiredLocalNote)] {
		t.Errorf("expected *.local.md inside retired skill directory excluded from InstalledFiles")
	}
	if installedSet[filepath.Clean(userCustomCmdFile)] {
		t.Errorf("expected user custom command excluded from InstalledFiles")
	}

	if _, err := Uninstall(env, cat, *legacy, backup.NewManager(env.BackupsDir()), false); err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}
	for _, wantGone := range []string{activeSkillFile, retiredToTicketsFile, retiredImplementFile, retiredCmdFile} {
		if _, err := os.Stat(wantGone); !os.IsNotExist(err) {
			t.Errorf("expected %s removed on Uninstall, err=%v", wantGone, err)
		}
	}
	if _, err := os.Stat(retiredLocalNote); err != nil {
		t.Errorf("expected %s preserved on Uninstall: %v", retiredLocalNote, err)
	}
	if _, err := os.Stat(userCustomCmdFile); err != nil {
		t.Errorf("expected %s preserved on Uninstall: %v", userCustomCmdFile, err)
	}
}
