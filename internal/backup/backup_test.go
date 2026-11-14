package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateListRestoreAndPrune(t *testing.T) {
	root := t.TempDir()
	backupsDir := filepath.Join(root, "backups")
	stateFile := filepath.Join(root, "state.json")
	initialState := `{"version":1,"receipts":[{"id":"claude:project","harness":"claude","scope":"project","track":"basics"}]}`
	if err := os.WriteFile(stateFile, []byte(initialState), 0600); err != nil {
		t.Fatal(err)
	}

	projDir := filepath.Join(root, "proj")
	if err := os.MkdirAll(filepath.Join(projDir, ".claude", "skills", "start"), 0755); err != nil {
		t.Fatal(err)
	}
	existingConfig := filepath.Join(projDir, "CLAUDE.md")
	if err := os.WriteFile(existingConfig, []byte("original claude config"), 0644); err != nil {
		t.Fatal(err)
	}
	dotfileTarget := filepath.Join(root, "dotfiles-gemini.md")
	if err := os.WriteFile(dotfileTarget, []byte("dotfile content"), 0644); err != nil {
		t.Fatal(err)
	}
	symlinkConfig := filepath.Join(projDir, "GEMINI.md")
	if err := os.Symlink(dotfileTarget, symlinkConfig); err != nil {
		t.Fatal(err)
	}
	newSkillFile := filepath.Join(projDir, ".claude", "skills", "start", "SKILL.md")
	localNote := filepath.Join(projDir, ".claude", "skills", "start", "notes.local.md")

	mgr := NewManager(backupsDir)
	mf, err := mgr.CreateSnapshot("claude", "project", "opinionated", []string{existingConfig, symlinkConfig, newSkillFile}, stateFile)
	if err != nil {
		t.Fatalf("CreateSnapshot error: %v", err)
	}
	if !mf.HasStateBackup {
		t.Errorf("expected HasStateBackup=true")
	}
	for _, entry := range mf.Entries {
		if strings.Contains(entry.BlobID, ":") || strings.Contains(entry.BlobID, "/") || strings.Contains(entry.BlobID, `\`) {
			t.Errorf("BlobID %q must be NTFS-safe indexed ID without colons or slashes", entry.BlobID)
		}
	}

	if err := os.WriteFile(existingConfig, []byte("overwritten claude config"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(symlinkConfig)
	if err := os.WriteFile(symlinkConfig, []byte("replaced symlink with regular file"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSkillFile, []byte("newly installed skill"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localNote, []byte("keep my local note"), 0644); err != nil {
		t.Fatal(err)
	}
	afterState := `{"version":1,"receipts":[{"id":"claude:project:` + filepath.ToSlash(projDir) + `/.claude","harness":"claude","scope":"project","root_dir":` + `"` + filepath.ToSlash(projDir) + `/.claude"` + `,"track":"opinionated"},{"id":"claude:project:/repoB/.claude","harness":"claude","scope":"project","root_dir":"/repoB/.claude","track":"product"},{"id":"zed:global","harness":"zed","scope":"global","track":"product"}]}`
	if err := os.WriteFile(stateFile, []byte(afterState), 0600); err != nil {
		t.Fatal(err)
	}

	res, err := mgr.RestoreSnapshot(mf.ID, stateFile)
	if err != nil {
		t.Fatalf("RestoreSnapshot error: %v", err)
	}
	if len(res.RestoredFiles) != 2 || len(res.RemovedFiles) != 1 || !res.StateRestored {
		t.Errorf("unexpected RestoreResult: %+v", res)
	}

	gotCfg, _ := os.ReadFile(existingConfig)
	if string(gotCfg) != "original claude config" {
		t.Errorf("ConfigFile not restored: %q", string(gotCfg))
	}
	linkInfo, err := os.Lstat(symlinkConfig)
	if err != nil || linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlinkConfig to be restored as symlink, got %v", err)
	}
	linkDest, _ := os.Readlink(symlinkConfig)
	if linkDest != dotfileTarget {
		t.Errorf("expected symlink target %q, got %q", dotfileTarget, linkDest)
	}
	if _, err := os.Stat(newSkillFile); !os.IsNotExist(err) {
		t.Errorf("expected newSkillFile to be removed on restore")
	}
	if _, err := os.Stat(localNote); err != nil {
		t.Errorf("expected localNote (*.local.md) to be preserved on restore: %v", err)
	}
	gotState, _ := os.ReadFile(stateFile)
	if !strings.Contains(string(gotState), `"track": "basics"`) || !strings.Contains(string(gotState), `"/repoB/.claude"`) || !strings.Contains(string(gotState), `"zed"`) {
		t.Errorf("expected claude repoA receipt restored to basics while preserving repoB and zed receipts, got:\n%s", string(gotState))
	}

	deepSymlinkParent := filepath.Join(root, "deep-link")
	if err := os.Symlink(projDir, deepSymlinkParent); err != nil {
		t.Fatal(err)
	}
	deepPath := filepath.Join(deepSymlinkParent, "a", "b", "c", "d", "file.md")
	if err := RejectSymlinkInChain(deepPath); err == nil {
		t.Errorf("expected RejectSymlinkInChain to catch symlinked ancestor 5 levels up")
	}

	for i := 0; i < 12; i++ {
		if _, err := mgr.CreateSnapshot("claude", "project", "basics", []string{existingConfig}, ""); err != nil {
			t.Fatal(err)
		}
	}
	snaps, err := mgr.ListSnapshots("claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 10 {
		t.Errorf("expected PruneOld to cap snapshots at 10, got %d", len(snaps))
	}

	if _, err := mgr.RestoreSnapshot("../evil", ""); err == nil {
		t.Errorf("expected path traversal snapshot ID to be rejected")
	}
}
