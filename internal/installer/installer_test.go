package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/backup"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/state"
)

func TestParseSkill_FullFrontmatter(t *testing.T) {
	content := []byte("---\nname: test-skill\ndescription: A test skill\nmodel: opus\ntags: [testing, quality]\nlanguages: [go, any]\n---\n# Test Skill")
	skill, err := parseSkill(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "test-skill" {
		t.Errorf("got name %q, want %q", skill.Name, "test-skill")
	}
	if skill.Model != "opus" {
		t.Errorf("got model %q, want %q", skill.Model, "opus")
	}
	if len(skill.Tags) != 2 {
		t.Errorf("got %d tags, want 2", len(skill.Tags))
	}
	if len(skill.Languages) != 2 {
		t.Errorf("got %d languages, want 2", len(skill.Languages))
	}
}

func TestParseSkill_MinimalFrontmatter(t *testing.T) {
	content := []byte("---\nname: my-skill\ndescription: Does things\n---\n# Content")
	skill, err := parseSkill(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "my-skill" {
		t.Errorf("got name %q, want %q", skill.Name, "my-skill")
	}
	if skill.Model != "" {
		t.Errorf("got model %q, want empty", skill.Model)
	}
}

func TestParseSkill_ExtraFields(t *testing.T) {
	content := []byte("---\nname: browser\ndescription: Browse\nallowed-tools: [Bash]\nargument-hint: URL\n---\n# Browser")
	skill, err := parseSkill(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "browser" {
		t.Errorf("got name %q, want %q", skill.Name, "browser")
	}
}

func TestParseSkill_MissingName(t *testing.T) {
	content := []byte("---\ndescription: no name\n---\n# Content")
	_, err := parseSkill(content)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseSkill_NoFrontmatter(t *testing.T) {
	content := []byte("# Just a markdown file\nNo frontmatter here")
	_, err := parseSkill(content)
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestDiscoverSkills(t *testing.T) {
	testFS := fstest.MapFS{
		"shared/skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: my-skill\ndescription: Test\n---\n# My Skill"),
		},
		"shared/skills/my-skill/references/guide.md": &fstest.MapFile{
			Data: []byte("# Reference"),
		},
		"shared/skills/another/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: another\ndescription: Another\n---\n# Another"),
		},
		"shared/skills/not-a-skill/readme.txt": &fstest.MapFile{
			Data: []byte("This has no SKILL.md"),
		},
	}

	inst := New(testFS, Options{})
	skills, err := inst.discoverSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	for _, s := range skills {
		if s.Model != "sonnet" {
			t.Errorf("skill %q: got model %q, want 'sonnet'", s.Name, s.Model)
		}
	}
}

func TestInstallSkills_CopiesDirectoryAndFilter(t *testing.T) {
	testFS := fstest.MapFS{
		"shared/skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: my-skill\ndescription: Test\ntags: [workflow]\n---\n# Content"),
		},
		"shared/skills/my-skill/references/helper.md": &fstest.MapFile{
			Data: []byte("# Helper"),
		},
		"shared/skills/untagged/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: untagged\ndescription: No tags\n---\n# Untagged"),
		},
	}

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, ".claude", "skills")

	inst := New(testFS, Options{})
	results, err := inst.InstallSkills(destDir, []string{"workflow"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(results), results)
	}
	if _, err := os.Stat(filepath.Join(destDir, "my-skill", "SKILL.md")); os.IsNotExist(err) {
		t.Errorf("SKILL.md not created")
	}
	if _, err := os.Stat(filepath.Join(destDir, "untagged", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("untagged skill should have been filtered out")
	}
}

func TestAdaptMarkdownForHarness_AllHarnesses(t *testing.T) {
	src := []byte("---\nname: demo\nmodel: opus\neffort: high\n---\nRead CLAUDE.md and ~/.claude/skills/demo and .claude/skills/other and .claude/agents/rev.md and TaskCreate.\n")

	claudeOut := string(AdaptMarkdownForHarness(src, HarnessClaude, false))
	if !strings.Contains(claudeOut, "model: opus") || !strings.Contains(claudeOut, "CLAUDE.md") {
		t.Errorf("expected claude markdown unchanged, got:\n%s", claudeOut)
	}

	agyOut := string(AdaptMarkdownForHarness(src, HarnessAntigravity, true))
	if strings.Contains(agyOut, "model: opus") || !strings.Contains(agyOut, "GEMINI.md") || !strings.Contains(agyOut, "~/.gemini/config/skills/demo") || !strings.Contains(agyOut, "task.md") {
		t.Errorf("unexpected antigravity markdown:\n%s", agyOut)
	}

	zedOut := string(AdaptMarkdownForHarness(src, HarnessZed, false))
	if strings.Contains(zedOut, "model: opus") || !strings.Contains(zedOut, "AGENTS.md") || !strings.Contains(zedOut, ".agents/skills/other") {
		t.Errorf("unexpected zed markdown:\n%s", zedOut)
	}

	ocOut := string(AdaptMarkdownForHarness(src, HarnessOpenCode, false))
	if strings.Contains(ocOut, "model: opus") || !strings.Contains(ocOut, "AGENTS.md") || !strings.Contains(ocOut, ".opencode/skills/other") || !strings.Contains(ocOut, ".opencode/agents/rev.md") {
		t.Errorf("unexpected opencode markdown:\n%s", ocOut)
	}
}

func TestInstallSkillsForHarness_AntigravityOverlayAndLocalPreservation(t *testing.T) {
	testFS := fstest.MapFS{
		"shared/skills/domain-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: domain-skill\ndescription: Domain skill\nmodel: opus\neffort: high\ntags: [core]\n---\nCheck ~/.claude/point-of-view and ~/.claude/skills/domain-skill and .claude/skills/other and .claude/agents/rev.md and CLAUDE.md and TaskCreate."),
		},
		"shared/skills/domain-skill/scripts/__pycache__/cached.pyc": &fstest.MapFile{
			Data: []byte("bytecode"),
		},
		"shared/skills/domain-skill/references/config.md": &fstest.MapFile{
			Data: []byte("secret config"),
		},
		"shared/skills/domain-skill/references/team.local.md": &fstest.MapFile{
			Data: []byte("secret team"),
		},
		"shared/skills/overridden-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: overridden-skill\ndescription: Base version\n---\n# Base"),
		},
		"shared/skills/overridden-skill/scripts/helper.py": &fstest.MapFile{
			Data: []byte("print('helper')\n"),
		},
		"shared/skills/overridden-skill/scripts/test_helper.py": &fstest.MapFile{
			Data: []byte("def test_x(): pass\n"),
		},
		"antigravity/skills/overridden-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: overridden-skill\ndescription: Antigravity version\n---\n# Overridden"),
		},
		"antigravity/skills/baton-pass/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: baton-pass\ndescription: Baton\n---\n# Baton"),
		},
	}

	agyDir := filepath.Join(t.TempDir(), "agy-skills")
	inst := New(testFS, Options{Force: true})
	if _, err := inst.InstallSkillsForHarness(agyDir, HarnessAntigravity, true, nil, nil); err != nil {
		t.Fatalf("InstallSkillsForHarness(antigravity) error: %v", err)
	}

	overriddenBytes, err := os.ReadFile(filepath.Join(agyDir, "overridden-skill", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(overriddenBytes), "Antigravity version") {
		t.Errorf("expected antigravity override to win")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "overridden-skill", "scripts", "helper.py")); os.IsNotExist(err) {
		t.Errorf("expected helper.py preserved")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "overridden-skill", "scripts", "test_helper.py")); !os.IsNotExist(err) {
		t.Errorf("expected test_helper.py skipped during install")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "baton-pass", "SKILL.md")); os.IsNotExist(err) {
		t.Errorf("expected baton-pass installed on antigravity")
	}

	zedDir := filepath.Join(t.TempDir(), "zed-skills")
	if _, err := inst.InstallSkillsForHarness(zedDir, HarnessZed, false, nil, nil); err != nil {
		t.Fatalf("InstallSkillsForHarness(zed) error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(zedDir, "baton-pass", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected baton-pass excluded on zed")
	}
}

func TestExecutePlan_TracksBackupWindowsHooksAndTrackSwitch(t *testing.T) {
	cat, err := catalog.Load(catalog.FS)
	if err != nil {
		t.Fatalf("catalog.Load error: %v", err)
	}

	tmpHome := t.TempDir()
	env := platform.Env{
		GOOS:    "windows",
		GOARCH:  "amd64",
		HomeDir: tmpHome,
		AppData: filepath.Join(tmpHome, "AppData", "Roaming"),
	}

	inst := New(catalog.FS, Options{Force: true})
	res1, err := inst.ExecutePlan(InstallPlan{
		Env:     env,
		Catalog: cat,
		Harness: HarnessAntigravity,
		Scope:   "global",
		TrackID: "opinionated",
		Backup:  true,
		Force:   true,
		Version: "4.0.0",
	})
	if err != nil {
		t.Fatalf("ExecutePlan(opinionated) error: %v", err)
	}
	if res1.BackupID == "" {
		t.Errorf("expected BackupID to be recorded")
	}
	hooksBytes, err := os.ReadFile(res1.Paths.HooksJSON)
	if err != nil {
		t.Fatalf("expected hooks.json created: %v", err)
	}
	if !strings.Contains(string(hooksBytes), "autonomous_breaker.py") {
		t.Errorf("expected Windows hooks.json to invoke autonomous_breaker.py, got:\n%s", string(hooksBytes))
	}

	res2, err := inst.ExecutePlan(InstallPlan{
		Env:     env,
		Catalog: cat,
		Harness: HarnessAntigravity,
		Scope:   "global",
		TrackID: "copy-polish",
		Backup:  true,
		Force:   true,
		Version: "4.0.0",
	})
	if err != nil {
		t.Fatalf("ExecutePlan(copy-polish) error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(res2.Paths.SkillsDir, "baton-pass", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected baton-pass removed after switching from opinionated to copy-polish")
	}
	if _, err := os.Stat(filepath.Join(res2.Paths.SkillsDir, "copy-edit", "SKILL.md")); err != nil {
		t.Errorf("expected copy-edit present in copy-polish track: %v", err)
	}
	if data, err := os.ReadFile(res2.Paths.HooksJSON); err == nil && strings.Contains(string(data), "autonomous-circuit-breaker") {
		t.Errorf("expected autonomous-circuit-breaker removed from hooks.json when switching to copy-polish track")
	}
}

func TestExecutePlan_LegacyUpgradePrunesRetiredV3SkillsAndBacksUp(t *testing.T) {
	v4FS := fstest.MapFS{
		"tracks.yaml": &fstest.MapFile{
			Data: []byte("version: \"4.0.0\"\ntracks:\n  - id: basics\n    name: Basics\n    skills: [plugin-updater]\n  - id: custom\n    name: Custom\n    skills: []\nskills:\n  - name: plugin-updater\n    category: workflow\n    slash_command: /plugin-update\n    short: Update\n    long: Update plugin\n    starter_prompt: /plugin-update\n  - name: brainstorming\n    category: product\n    slash_command: /brainstorming\n    short: Brainstorm\n    long: Brainstorm ideas\n    starter_prompt: /brainstorming\n"),
		},
		"shared/skills/plugin-updater/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: plugin-updater\ndescription: Update\n---\n# Updater\n"),
		},
		"shared/skills/brainstorming/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: brainstorming\ndescription: Brainstorm\n---\n# Brainstorming\n"),
		},
		"claude/templates/CLAUDE-BASE.md": &fstest.MapFile{
			Data: []byte("# Project Config\n"),
		},
		"claude/commands/plugin-update.md": &fstest.MapFile{
			Data: []byte("# /plugin-update\n"),
		},
		"claude/commands/config/show.md": &fstest.MapFile{
			Data: []byte("# /config:show\n"),
		},
	}
	cat, err := catalog.Load(v4FS)
	if err != nil {
		t.Fatalf("catalog.Load error: %v", err)
	}

	tmpHome := t.TempDir()
	env := platform.Env{
		GOOS:    "darwin",
		GOARCH:  "arm64",
		HomeDir: tmpHome,
	}
	projDir := filepath.Join(tmpHome, "legacy-v3-proj")
	paths, err := env.ResolvePaths(HarnessClaude, "project", projDir)
	if err != nil {
		t.Fatal(err)
	}

	legacySkill := filepath.Join(paths.SkillsDir, "to-tickets", "SKILL.md")
	legacyCmd := filepath.Join(paths.CommandsDir, "to-tickets.md")
	if err := os.MkdirAll(filepath.Dir(legacySkill), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(legacyCmd), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacySkill, []byte("# Legacy to-tickets"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyCmd, []byte("# /to-tickets"), 0644); err != nil {
		t.Fatal(err)
	}

	inst := New(v4FS, Options{Force: true})
	res, err := inst.ExecutePlan(InstallPlan{
		Env:        env,
		Catalog:    cat,
		Harness:    HarnessClaude,
		Scope:      "project",
		ProjectDir: projDir,
		TrackID:    "basics",
		Backup:     true,
		Force:      true,
		Version:    "4.0.0",
	})
	if err != nil {
		t.Fatalf("ExecutePlan error: %v", err)
	}
	if res.BackupID == "" {
		t.Fatalf("expected BackupID to be recorded")
	}
	if _, err := os.Stat(legacySkill); !os.IsNotExist(err) {
		t.Errorf("expected retired v3 skill %s pruned during upgrade, err=%v", legacySkill, err)
	}
	if _, err := os.Stat(legacyCmd); !os.IsNotExist(err) {
		t.Errorf("expected retired v3 command %s pruned during upgrade, err=%v", legacyCmd, err)
	}

	mgr := backup.NewManager(env.BackupsDir())
	if _, err := mgr.RestoreSnapshot(res.BackupID, env.StateFilePath()); err != nil {
		t.Fatalf("RestoreSnapshot error: %v", err)
	}
	if got, err := os.ReadFile(legacySkill); err != nil || string(got) != "# Legacy to-tickets" {
		t.Errorf("expected safety backup to restore %s, got %q, err=%v", legacySkill, string(got), err)
	}
	if got, err := os.ReadFile(legacyCmd); err != nil || string(got) != "# /to-tickets" {
		t.Errorf("expected safety backup to restore %s, got %q, err=%v", legacyCmd, string(got), err)
	}

	customProj := filepath.Join(tmpHome, "custom-proj")
	customRes, err := inst.ExecutePlan(InstallPlan{
		Env:          env,
		Catalog:      cat,
		Harness:      HarnessClaude,
		Scope:        "project",
		ProjectDir:   customProj,
		TrackID:      "custom",
		CustomSkills: []string{"to-tickets", "plugin-updater"},
		Force:        true,
		Version:      "4.0.0",
	})
	if err != nil {
		t.Fatalf("ExecutePlan(custom) error: %v", err)
	}
	if len(customRes.Receipt.Skills) != 1 || customRes.Receipt.Skills[0] != "plugin-updater" {
		t.Errorf("expected retired skill dropped from custom Receipt.Skills, got %v", customRes.Receipt.Skills)
	}

	allRetiredProj := filepath.Join(tmpHome, "all-retired-custom-proj")
	allRetiredRes, err := inst.ExecutePlan(InstallPlan{
		Env:          env,
		Catalog:      cat,
		Harness:      HarnessClaude,
		Scope:        "project",
		ProjectDir:   allRetiredProj,
		TrackID:      "custom",
		CustomSkills: []string{"to-tickets", "../plugin-updater", `..\plugin-updater`, ".", ".."},
		Force:        true,
		Version:      "4.0.0",
	})
	if err != nil {
		t.Fatalf("ExecutePlan(all-retired custom) error: %v", err)
	}
	if len(allRetiredRes.Skills) != 0 || len(allRetiredRes.Receipt.Skills) != 0 {
		t.Errorf("expected empty Skills when all custom skills are retired/invalid, got %v / %v", allRetiredRes.Skills, allRetiredRes.Receipt.Skills)
	}
	if _, err := os.Stat(filepath.Join(allRetiredRes.Paths.SkillsDir, "brainstorming", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected brainstorming NOT installed when selectedSkills is empty, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(allRetiredRes.Paths.SkillsDir, "plugin-updater", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected path-traversal CustomSkills rejected and plugin-updater NOT installed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(allRetiredRes.Paths.CommandsDir, "config", "show.md")); !os.IsNotExist(err) {
		t.Errorf("expected subdir command config/show.md NOT installed when selectedSkills is empty, err=%v", err)
	}
}

func TestMergeConfigContent_IdempotentAndPreservesUserRules(t *testing.T) {
	userRules := "# My Custom Project Rules\n\n- Always run `npm run check`.\r\n"
	incoming1 := "# Workflow Rules v1\n\nPhase 1 -> Phase 12"
	merged1 := MergeConfigContent([]byte(userRules), []byte(incoming1))
	s1 := string(merged1)
	if !strings.Contains(s1, "# My Custom Project Rules") || !strings.Contains(s1, "Always run `npm run check`") {
		t.Fatalf("expected user rules preserved in merged output:\n%s", s1)
	}
	if !HasManagedWorkflowSection(merged1) || !strings.Contains(s1, "Phase 1 -> Phase 12") {
		t.Fatalf("expected managed workflow section in merged output:\n%s", s1)
	}

	incoming2 := "# Workflow Rules v2\n\nUpdated phases"
	merged2 := MergeConfigContent(merged1, []byte(incoming2))
	s2 := string(merged2)
	if strings.Count(s2, ManagedWorkflowBegin) != 1 || strings.Count(s2, ManagedWorkflowEnd) != 1 {
		t.Fatalf("expected exactly 1 managed section after second merge, got:\n%s", s2)
	}
	if strings.Contains(s2, "Workflow Rules v1") || !strings.Contains(s2, "Workflow Rules v2") {
		t.Fatalf("expected v1 replaced by v2 in-place, got:\n%s", s2)
	}

	stripped, ok := StripManagedWorkflowSection(merged2)
	if !ok {
		t.Fatalf("expected StripManagedWorkflowSection to return true")
	}
	if HasManagedWorkflowSection(stripped) || !strings.Contains(string(stripped), "# My Custom Project Rules") {
		t.Fatalf("expected only user rules remaining after strip, got:\n%s", string(stripped))
	}

	freshMerged := MergeConfigContent(nil, []byte(incoming1))
	emptyAfterStrip, ok := StripManagedWorkflowSection(freshMerged)
	if !ok || len(emptyAfterStrip) != 0 {
		t.Fatalf("expected empty content when stripping file created purely from managed section, got %q (ok=%v)", string(emptyAfterStrip), ok)
	}
}

func TestExecutePlan_ConfigModeMergeReplaceSkipAndUninstallStrip(t *testing.T) {
	cat, err := catalog.Load(catalog.FS)
	if err != nil {
		t.Fatal(err)
	}
	tmpHome := t.TempDir()
	projDir := filepath.Join(tmpHome, "repo")
	if err := os.MkdirAll(projDir, 0755); err != nil {
		t.Fatal(err)
	}
	env := platform.Env{GOOS: "darwin", GOARCH: "arm64", HomeDir: tmpHome}

	claudeMD := filepath.Join(projDir, "CLAUDE.md")
	originalUserContent := "# Team Rules\n\nKeep domain logic pure.\n"
	if err := os.WriteFile(claudeMD, []byte(originalUserContent), 0644); err != nil {
		t.Fatal(err)
	}

	inst := New(catalog.FS, Options{Force: false})
	resMerge, err := inst.ExecutePlan(InstallPlan{
		Env:        env,
		Catalog:    cat,
		Harness:    HarnessClaude,
		Scope:      "project",
		ProjectDir: projDir,
		TrackID:    "basics",
		ConfigMode: ConfigModeMerge,
		Backup:     true,
		Force:      false,
		Version:    "4.0.0",
	})
	if err != nil {
		t.Fatalf("ExecutePlan(ConfigModeMerge) error: %v", err)
	}
	if resMerge.ConfigAction != "merged" {
		t.Errorf("ConfigAction = %q, want merged", resMerge.ConfigAction)
	}
	gotBytes, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gotBytes), "Keep domain logic pure.") || !HasManagedWorkflowSection(gotBytes) {
		t.Fatalf("expected user content + managed section in CLAUDE.md, got:\n%s", string(gotBytes))
	}

	postInstallEdit := string(gotBytes) + "\n## Added After Install\n- Post-install note\n"
	if err := os.WriteFile(claudeMD, []byte(postInstallEdit), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := backup.NewManager(env.BackupsDir())
	unRes, err := state.Uninstall(env, cat, resMerge.Receipt, mgr, true)
	if err != nil {
		t.Fatalf("state.Uninstall error: %v", err)
	}
	if unRes.RestoredConfig != claudeMD {
		t.Errorf("expected RestoredConfig=%q, got %q", claudeMD, unRes.RestoredConfig)
	}
	afterUninstall, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatal(err)
	}
	if HasManagedWorkflowSection(afterUninstall) {
		t.Errorf("expected managed workflow section stripped on uninstall, got:\n%s", string(afterUninstall))
	}
	if !strings.Contains(string(afterUninstall), "Keep domain logic pure.") || !strings.Contains(string(afterUninstall), "Post-install note") {
		t.Errorf("expected both pre- and post-install user rules preserved on uninstall, got:\n%s", string(afterUninstall))
	}

	resGlobalMerge, err := inst.ExecutePlan(InstallPlan{
		Env:        env,
		Catalog:    cat,
		Harness:    HarnessAntigravity,
		Scope:      "global",
		TrackID:    "basics",
		ConfigMode: ConfigModeMerge,
		Backup:     false,
		Force:      false,
		Version:    "4.0.0",
	})
	if err != nil {
		t.Fatalf("ExecutePlan(global Antigravity ConfigModeMerge) error: %v", err)
	}
	if resGlobalMerge.ConfigAction != "created" {
		t.Errorf("expected ConfigAction=created on fresh global GEMINI.md, got %q", resGlobalMerge.ConfigAction)
	}
}
