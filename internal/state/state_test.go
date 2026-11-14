package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
}
