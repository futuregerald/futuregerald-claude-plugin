package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/state"
)

type fakeRunner struct {
	handlers map[string][]byte
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	if out, ok := f.handlers[key]; ok {
		return out, nil
	}
	return nil, fmt.Errorf("unexpected command: %s", key)
}

func buildRepoTarball(t *testing.T, extraEntries []tar.Header, extraBodies [][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	addFile := func(name string, body string) {
		hdr := &tar.Header{
			Name:     name,
			Typeflag: tar.TypeReg,
			Mode:     0644,
			Size:     int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}

	paxHdr := &tar.Header{
		Name:     "pax_global_header",
		Typeflag: tar.TypeXGlobalHeader,
	}
	_ = tw.WriteHeader(paxHdr)

	prefix := "futuregerald-futuregerald-claude-plugin-abc1234/"
	tracksYAML := `version: "4.0.0"
tracks:
  - id: basics
    name: "Just the Basics"
    install_agents: false
    install_hooks: false
    lightweight_config: true
    skills:
      - plugin-updater
  - id: custom
    name: "Build Your Own"
    install_agents: false
    install_hooks: false
    lightweight_config: true
    skills: []
skills:
  - name: plugin-updater
    category: workflow
    slash_command: "/plugin-update"
    short: "Update plugin"
    long: "Long description"
    starter_prompt: "/plugin-update"
  - name: brainstorming
    category: product
    slash_command: "/brainstorming"
    short: "Brainstorm"
    long: "Long brainstorm description"
    starter_prompt: "/brainstorming"
`
	addFile(prefix+"internal/catalog/tracks.yaml", tracksYAML)
	addFile(prefix+"internal/catalog/shared/skills/plugin-updater/SKILL.md", "---\nname: plugin-updater\ndescription: Updated skill body\n---\n# Updated Plugin Skill\n")
	addFile(prefix+"internal/catalog/shared/skills/brainstorming/SKILL.md", "---\nname: brainstorming\ndescription: Brainstorming skill\n---\n# Brainstorming\n")
	addFile(prefix+"internal/catalog/claude/templates/CLAUDE-BASE.md", "# Claude Base\n")
	addFile(prefix+"internal/catalog/claude/CLAUDE-LIGHT.md", "# Claude Light\n")
	addFile(prefix+"internal/catalog/claude/commands/plugin-update.md", "# /plugin-update\n")

	for idx, hdr := range extraEntries {
		h := hdr
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if idx < len(extraBodies) && len(extraBodies[idx]) > 0 {
			if _, err := tw.Write(extraBodies[idx]); err != nil {
				t.Fatal(err)
			}
		}
	}

	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestCheckAndApplyWithTarballSecurity(t *testing.T) {
	tarball := buildRepoTarball(t, nil, nil)
	runner := fakeRunner{
		handlers: map[string][]byte{
			"gh api repos/futuregerald/futuregerald-claude-plugin/commits/main":            []byte(`{"sha":"f9e8d7c6","commit":{"message":"feat: update plugin-updater\n\ndetails","committer":{"date":"2026-11-14T00:00:00Z"}}}`),
			"gh api repos/futuregerald/futuregerald-claude-plugin/compare/a1b2c3d4...main": []byte(`{"commits":[{"sha":"f9e8d7c6","commit":{"message":"feat: update plugin-updater"}}],"files":[{"filename":"internal/catalog/shared/skills/plugin-updater/SKILL.md"}]}`),
			"gh api repos/futuregerald/futuregerald-claude-plugin/releases/latest":         []byte(`{"tag_name":"v4.1.0"}`),
			"gh api repos/futuregerald/futuregerald-claude-plugin/tarball/main":            tarball,
		},
	}

	tmp := t.TempDir()
	env := platform.Env{GOOS: "darwin", GOARCH: "arm64", HomeDir: tmp}
	projDir := filepath.Join(tmp, "proj")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	userClaudeMD := filepath.Join(projDir, "CLAUDE.md")
	if err := os.WriteFile(userClaudeMD, []byte("# Custom User Project Config"), 0644); err != nil {
		t.Fatal(err)
	}

	receipt := state.Receipt{
		Harness:   "claude",
		Scope:     "project",
		RootDir:   projDir,
		Track:     "basics",
		CommitSHA: "a1b2c3d4",
	}

	rep, err := Check(context.Background(), runner, receipt, "4.0.0")
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}
	if !rep.UpdateAvailable || !rep.BinaryUpdate || rep.LatestSHA != "f9e8d7c6" {
		t.Errorf("unexpected CheckReport: %+v", rep)
	}
	if len(rep.ChangedSkills) != 1 || rep.ChangedSkills[0] != "plugin-updater" {
		t.Errorf("expected changed_skills=[plugin-updater], got %v", rep.ChangedSkills)
	}

	res, sha, err := Apply(context.Background(), runner, env, receipt, ApplyOptions{
		Backup:         true,
		Force:          true,
		CurrentVersion: "4.0.0",
	})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if sha != "f9e8d7c6" || res.Receipt.CommitSHA != "f9e8d7c6" {
		t.Errorf("expected CommitSHA updated to f9e8d7c6, got %q", res.Receipt.CommitSHA)
	}
	gotClaudeMD, _ := os.ReadFile(userClaudeMD)
	if string(gotClaudeMD) != "# Custom User Project Config" {
		t.Errorf("expected Apply to preserve customized project CLAUDE.md, got %q", string(gotClaudeMD))
	}
	skillBytes, err := os.ReadFile(filepath.Join(projDir, ".claude", "skills", "plugin-updater", "SKILL.md"))
	if err != nil {
		t.Fatalf("expected updated skill written: %v", err)
	}
	if !strings.Contains(string(skillBytes), "Updated Plugin Skill") {
		t.Errorf("unexpected skill content: %s", string(skillBytes))
	}

	symlinkTar := buildRepoTarball(t, []tar.Header{
		{
			Name:     "futuregerald-futuregerald-claude-plugin-abc1234/evil-link",
			Typeflag: tar.TypeSymlink,
			Linkname: "/etc/passwd",
		},
	}, nil)
	if err := ExtractRepoTarball(bytes.NewReader(symlinkTar), t.TempDir()); err == nil {
		t.Errorf("expected symlink in tarball to be rejected")
	}

	traversalTar := buildRepoTarball(t, []tar.Header{
		{
			Name:     "futuregerald-futuregerald-claude-plugin-abc1234/../../escape.txt",
			Typeflag: tar.TypeReg,
			Size:     4,
		},
	}, [][]byte{[]byte("boom")})
	if err := ExtractRepoTarball(bytes.NewReader(traversalTar), t.TempDir()); err == nil {
		t.Errorf("expected .. traversal in tarball to be rejected")
	}
}

func TestSelfUpgradeBinary_WindowsRenameAndChecksum(t *testing.T) {
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	fw, err := zw.Create("skill-installer.exe")
	if err != nil {
		t.Fatal(err)
	}
	newBinContent := []byte("NEW_WINDOWS_BINARY_V4_1_0")
	if _, err := fw.Write(newBinContent); err != nil {
		t.Fatal(err)
	}
	_ = zw.Close()
	zipBytes := zipBuf.Bytes()

	sum := sha256.Sum256(zipBytes)
	hashHex := hex.EncodeToString(sum[:])
	archiveName := "skill-installer_4.1.0_windows_amd64.zip"
	checksumsTxt := fmt.Sprintf("%s  %s\n", hashHex, archiveName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksumsTxt))
		case "/" + archiveName:
			_, _ = w.Write(zipBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmp := t.TempDir()
	exePath := filepath.Join(tmp, "skill-installer.exe")
	if err := os.WriteFile(exePath, []byte("OLD_EXE"), 0755); err != nil {
		t.Fatal(err)
	}

	winEnv := platform.Env{GOOS: "windows", GOARCH: "amd64", HomeDir: tmp}
	ver, err := SelfUpgradeBinary(context.Background(), winEnv, exePath, "v4.1.0", srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("SelfUpgradeBinary error: %v", err)
	}
	if ver != "4.1.0" {
		t.Errorf("ver = %q, want 4.1.0", ver)
	}
	gotNew, _ := os.ReadFile(exePath)
	if string(gotNew) != string(newBinContent) {
		t.Errorf("exePath was not updated: %q", string(gotNew))
	}
	gotOld, err := os.ReadFile(exePath + ".old")
	if err != nil || string(gotOld) != "OLD_EXE" {
		t.Errorf("expected Windows .exe.old backup preserved during active run: %v, %q", err, string(gotOld))
	}
}

func TestApply_CustomTrackDropsRetiredSkillsFromReceipt(t *testing.T) {
	tarball := buildRepoTarball(t, nil, nil)
	runner := fakeRunner{
		handlers: map[string][]byte{
			"gh api repos/futuregerald/futuregerald-claude-plugin/commits/main": []byte(`{"sha":"c0ffee12","commit":{"message":"chore: prune skills","committer":{"date":"2026-11-14T00:00:00Z"}}}`),
			"gh api repos/futuregerald/futuregerald-claude-plugin/tarball/main": tarball,
		},
	}

	tmp := t.TempDir()
	env := platform.Env{GOOS: "darwin", GOARCH: "arm64", HomeDir: tmp}
	projDir := filepath.Join(tmp, "custom-proj")
	paths, err := env.ResolvePaths("claude", "project", projDir)
	if err != nil {
		t.Fatal(err)
	}

	retiredSkillFile := filepath.Join(paths.SkillsDir, "to-tickets", "SKILL.md")
	retiredCmdFile := filepath.Join(paths.CommandsDir, "to-tickets.md")
	if err := os.MkdirAll(filepath.Dir(retiredSkillFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(retiredCmdFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retiredSkillFile, []byte("# old to-tickets"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retiredCmdFile, []byte("# /to-tickets"), 0644); err != nil {
		t.Fatal(err)
	}

	receipt := state.Receipt{
		ID:             state.ReceiptKey("claude", "project", paths.RootDir),
		Harness:        "claude",
		Scope:          "project",
		RootDir:        paths.RootDir,
		Track:          "custom",
		Skills:         []string{"to-tickets", "brainstorming"},
		InstalledFiles: []string{retiredSkillFile, retiredCmdFile},
		CommitSHA:      "a1b2c3d4",
	}
	store := state.NewStore(env.StateFilePath())
	if err := store.Upsert(receipt); err != nil {
		t.Fatal(err)
	}

	res, _, err := Apply(context.Background(), runner, env, receipt, ApplyOptions{
		Backup:         true,
		Force:          true,
		CurrentVersion: "4.0.0",
	})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if len(res.Receipt.Skills) != 1 || res.Receipt.Skills[0] != "brainstorming" {
		t.Errorf("expected Receipt.Skills=[brainstorming], got %v", res.Receipt.Skills)
	}
	if _, err := os.Stat(retiredSkillFile); !os.IsNotExist(err) {
		t.Errorf("expected retired skill file pruned on custom update, err=%v", err)
	}
	if _, err := os.Stat(retiredCmdFile); !os.IsNotExist(err) {
		t.Errorf("expected retired command file pruned on custom update, err=%v", err)
	}

	projAllRetired := filepath.Join(tmp, "all-retired-proj")
	pathsAllRetired, err := env.ResolvePaths("claude", "project", projAllRetired)
	if err != nil {
		t.Fatal(err)
	}
	allRetiredReceipt := state.Receipt{
		ID:        state.ReceiptKey("claude", "project", pathsAllRetired.RootDir),
		Harness:   "claude",
		Scope:     "project",
		RootDir:   pathsAllRetired.RootDir,
		Track:     "custom",
		Skills:    []string{"to-tickets"},
		CommitSHA: "a1b2c3d4",
	}
	if err := store.Upsert(allRetiredReceipt); err != nil {
		t.Fatal(err)
	}
	resFallback, _, err := Apply(context.Background(), runner, env, allRetiredReceipt, ApplyOptions{
		Backup:         false,
		Force:          true,
		CurrentVersion: "4.0.0",
	})
	if err != nil {
		t.Fatalf("Apply(all retired) error: %v", err)
	}
	if len(resFallback.Receipt.Skills) != 1 || resFallback.Receipt.Skills[0] != "plugin-updater" {
		t.Errorf("expected fallback Receipt.Skills=[plugin-updater], got %v", resFallback.Receipt.Skills)
	}
	if _, err := os.Stat(filepath.Join(pathsAllRetired.SkillsDir, "brainstorming", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected brainstorming NOT installed when all custom skills retired, err=%v", err)
	}

	projLegacyNilSkills := filepath.Join(tmp, "legacy-nil-skills-proj")
	pathsLegacyNil, err := env.ResolvePaths("claude", "project", projLegacyNilSkills)
	if err != nil {
		t.Fatal(err)
	}
	legacyOnlyRetired := filepath.Join(pathsLegacyNil.SkillsDir, "to-tickets", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(legacyOnlyRetired), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyOnlyRetired, []byte("# legacy only to-tickets"), 0644); err != nil {
		t.Fatal(err)
	}
	legacyNilReceipt := state.Receipt{
		ID:             state.ReceiptKey("claude", "project", pathsLegacyNil.RootDir),
		Harness:        "claude",
		Scope:          "project",
		RootDir:        pathsLegacyNil.RootDir,
		Track:          "custom",
		Skills:         nil,
		InstalledFiles: []string{legacyOnlyRetired},
		CommitSHA:      "a1b2c3d4",
	}
	if err := store.Upsert(legacyNilReceipt); err != nil {
		t.Fatal(err)
	}
	resLegacyNil, _, err := Apply(context.Background(), runner, env, legacyNilReceipt, ApplyOptions{
		Backup:         false,
		Force:          true,
		CurrentVersion: "4.0.0",
	})
	if err != nil {
		t.Fatalf("Apply(legacy nil skills) error: %v", err)
	}
	if len(resLegacyNil.Receipt.Skills) != 1 || resLegacyNil.Receipt.Skills[0] != "plugin-updater" {
		t.Errorf("expected legacy nil-skills fallback Receipt.Skills=[plugin-updater], got %v", resLegacyNil.Receipt.Skills)
	}
	if _, err := os.Stat(legacyOnlyRetired); !os.IsNotExist(err) {
		t.Errorf("expected legacy retired skill pruned, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(pathsLegacyNil.SkillsDir, "brainstorming", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected brainstorming NOT installed on legacy nil-skills custom update, err=%v", err)
	}

	projMergedCfg := filepath.Join(tmp, "merged-cfg-proj")
	pathsMergedCfg, err := env.ResolvePaths("claude", "project", projMergedCfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projMergedCfg, 0755); err != nil {
		t.Fatal(err)
	}
	initialMerged := state.MergeConfigContent([]byte("# Custom Team Rules\n- Keep handlers thin\n"), []byte("# Old Workflow Rules"))
	if err := os.WriteFile(pathsMergedCfg.ConfigFile, initialMerged, 0644); err != nil {
		t.Fatal(err)
	}
	mergedReceipt := state.Receipt{
		ID:        state.ReceiptKey("claude", "project", pathsMergedCfg.RootDir),
		Harness:   "claude",
		Scope:     "project",
		RootDir:   pathsMergedCfg.RootDir,
		Track:     "basics",
		Skills:    []string{"brainstorming", "plugin-updater"},
		CommitSHA: "a1b2c3d4",
	}
	if _, _, err := Apply(context.Background(), runner, env, mergedReceipt, ApplyOptions{
		Backup:         false,
		Force:          true,
		CurrentVersion: "4.0.0",
	}); err != nil {
		t.Fatalf("Apply(merged project config) error: %v", err)
	}
	updatedMergedBytes, err := os.ReadFile(pathsMergedCfg.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updatedMergedBytes), "Keep handlers thin") || strings.Contains(string(updatedMergedBytes), "# Old Workflow Rules") {
		t.Errorf("expected Apply to preserve custom rules and update managed section in project CLAUDE.md, got:\n%s", string(updatedMergedBytes))
	}
}
