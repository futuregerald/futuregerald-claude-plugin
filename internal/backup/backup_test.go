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
	if err := os.WriteFile(stateFile, []byte(`{"version":"before"}`), 0600); err != nil {
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
	newSkillFile := filepath.Join(projDir, ".claude", "skills", "start", "SKILL.md")
	localNote := filepath.Join(projDir, ".claude", "skills", "start", "notes.local.md")

	mgr := NewManager(backupsDir)
	mf, err := mgr.CreateSnapshot("claude", "project", "opinionated", []string{existingConfig, newSkillFile}, stateFile)
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
	if err := os.WriteFile(newSkillFile, []byte("newly installed skill"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localNote, []byte("keep my local note"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte(`{"version":"after"}`), 0600); err != nil {
		t.Fatal(err)
	}

	res, err := mgr.RestoreSnapshot(mf.ID, stateFile)
	if err != nil {
		t.Fatalf("RestoreSnapshot error: %v", err)
	}
	if len(res.RestoredFiles) != 1 || len(res.RemovedFiles) != 1 || !res.StateRestored {
		t.Errorf("unexpected RestoreResult: %+v", res)
	}

	gotCfg, _ := os.ReadFile(existingConfig)
	if string(gotCfg) != "original claude config" {
		t.Errorf("ConfigFile not restored: %q", string(gotCfg))
	}
	if _, err := os.Stat(newSkillFile); !os.IsNotExist(err) {
		t.Errorf("expected newSkillFile to be removed on restore")
	}
	if _, err := os.Stat(localNote); err != nil {
		t.Errorf("expected localNote (*.local.md) to be preserved on restore: %v", err)
	}
	gotState, _ := os.ReadFile(stateFile)
	if string(gotState) != `{"version":"before"}` {
		t.Errorf("state.json not restored: %q", string(gotState))
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
