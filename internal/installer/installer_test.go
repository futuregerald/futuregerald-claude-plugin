package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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
		"skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: my-skill\ndescription: Test\n---\n# My Skill"),
		},
		"skills/my-skill/references/guide.md": &fstest.MapFile{
			Data: []byte("# Reference"),
		},
		"skills/another/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: another\ndescription: Another\n---\n# Another"),
		},
		"skills/not-a-skill/readme.txt": &fstest.MapFile{
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

	// Verify default model is set
	for _, s := range skills {
		if s.Model != "sonnet" {
			t.Errorf("skill %q: got model %q, want 'sonnet'", s.Name, s.Model)
		}
	}
}

func TestDiscoverSkills_LowercaseFallback(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/lower/skill.md": &fstest.MapFile{
			Data: []byte("---\nname: lower\ndescription: Lowercase\n---\n# Lower"),
		},
	}

	inst := New(testFS, Options{})
	skills, err := inst.discoverSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "lower" {
		t.Errorf("got name %q, want 'lower'", skills[0].Name)
	}
}

func TestDiscoverSkills_ParseFailureFallback(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/no-frontmatter/SKILL.md": &fstest.MapFile{
			Data: []byte("# Just content, no frontmatter"),
		},
	}

	inst := New(testFS, Options{})
	skills, err := inst.discoverSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	// Should fall back to directory name
	if skills[0].Name != "no-frontmatter" {
		t.Errorf("got name %q, want 'no-frontmatter'", skills[0].Name)
	}
}

func TestInstallSkills_CopiesDirectory(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: my-skill\ndescription: Test\n---\n# Content"),
		},
		"skills/my-skill/references/helper.md": &fstest.MapFile{
			Data: []byte("# Helper"),
		},
	}

	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, ".claude", "skills")

	inst := New(testFS, Options{})
	results, err := inst.InstallSkills(destDir, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(results), results)
	}

	// Verify SKILL.md was created
	skillPath := filepath.Join(destDir, "my-skill", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("SKILL.md not created at %s", skillPath)
	}

	// Verify reference was created
	refPath := filepath.Join(destDir, "my-skill", "references", "helper.md")
	if _, err := os.Stat(refPath); os.IsNotExist(err) {
		t.Errorf("reference not created at %s", refPath)
	}
}

func TestInstallSkills_FilterByTags(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/tagged/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: tagged\ndescription: Has tags\ntags: [workflow, core]\n---\n# Tagged"),
		},
		"skills/untagged/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: untagged\ndescription: No tags\n---\n# Untagged"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})

	results, err := inst.InstallSkills(tmpDir, []string{"workflow"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}

	if !strings.Contains(results[0], "tagged") {
		t.Errorf("expected tagged skill installed, got: %s", results[0])
	}
}

func TestInstallSkills_NoFilter(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/a/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: a\ndescription: Skill A\n---\n# A"),
		},
		"skills/b/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: b\ndescription: Skill B\n---\n# B"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})

	results, err := inst.InstallSkills(tmpDir, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestInstallAgents_DefaultNaming(t *testing.T) {
	testFS := fstest.MapFS{
		"agents/debugger.md": &fstest.MapFile{
			Data: []byte("---\nname: debugger\ndescription: Debug\n---\n# Debugger"),
		},
		"agents/reviewer.md": &fstest.MapFile{
			Data: []byte("---\nname: reviewer\ndescription: Review\n---\n# Reviewer"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})
	results, err := inst.InstallAgents(tmpDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "debugger.md")); os.IsNotExist(err) {
		t.Error("debugger.md not created")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "reviewer.md")); os.IsNotExist(err) {
		t.Error("reviewer.md not created")
	}
}

func TestInstallAgents_CopilotNaming(t *testing.T) {
	testFS := fstest.MapFS{
		"agents/debugger.md": &fstest.MapFile{
			Data: []byte("---\nname: debugger\ndescription: Debug\n---\n# Debugger"),
		},
		"agents/reviewer.md": &fstest.MapFile{
			Data: []byte("---\nname: reviewer\ndescription: Review\n---\n# Reviewer"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})
	results, err := inst.InstallAgents(tmpDir, CopilotAgentName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "debugger.agent.md")); os.IsNotExist(err) {
		t.Error("debugger.agent.md not created")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "reviewer.agent.md")); os.IsNotExist(err) {
		t.Error("reviewer.agent.md not created")
	}
}

func TestCopilotAgentName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"debugger.md", "debugger.agent.md"},
		{"code-reviewer.md", "code-reviewer.agent.md"},
	}
	for _, tt := range tests {
		got := CopilotAgentName(tt.in)
		if got != tt.want {
			t.Errorf("CopilotAgentName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestInstallAgents_NoAgentsDir(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/a/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: a\ndescription: A\n---\n# A"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})
	results, err := inst.InstallAgents(tmpDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestInstallCommands(t *testing.T) {
	testFS := fstest.MapFS{
		"commands/init-claude-md/COMMAND.md": &fstest.MapFile{
			Data: []byte("# Init Command"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})
	results, err := inst.InstallCommands(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	cmdPath := filepath.Join(tmpDir, "init-claude-md", "COMMAND.md")
	if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
		t.Errorf("COMMAND.md not created at %s", cmdPath)
	}
}

func TestInstallCommands_NoCommandsDir(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/a/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: a\ndescription: A\n---\n# A"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})
	results, err := inst.InstallCommands(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		options    Options
		setupFile  bool
		wantPrefix string
		wantWrite  bool
	}{
		{
			name:       "create new file",
			options:    Options{},
			setupFile:  false,
			wantPrefix: "CREATED:",
			wantWrite:  true,
		},
		{
			name:       "skip existing file without force",
			options:    Options{Force: false},
			setupFile:  true,
			wantPrefix: "SKIP:",
			wantWrite:  false,
		},
		{
			name:       "overwrite with force",
			options:    Options{Force: true},
			setupFile:  true,
			wantPrefix: "UPDATED:",
			wantWrite:  true,
		},
		{
			name:       "dry run new file",
			options:    Options{DryRun: true},
			setupFile:  false,
			wantPrefix: "WOULD CREATE:",
			wantWrite:  false,
		},
		{
			name:       "dry run existing file with force",
			options:    Options{Force: true, DryRun: true},
			setupFile:  true,
			wantPrefix: "WOULD OVERWRITE:",
			wantWrite:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".txt")

			if tt.setupFile {
				if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
					t.Fatalf("Failed to create setup file: %v", err)
				}
			}

			inst := &Installer{options: tt.options}
			result, err := inst.writeFile(testFile, []byte("new content"))

			if err != nil {
				t.Fatalf("writeFile() error: %v", err)
			}

			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("writeFile() result = %q, want prefix %q", result, tt.wantPrefix)
			}

			content, readErr := os.ReadFile(testFile)
			if tt.wantWrite {
				if readErr != nil {
					t.Errorf("Expected file to exist after write")
				} else if string(content) != "new content" {
					t.Errorf("File content = %q, want %q", string(content), "new content")
				}
			} else if tt.setupFile {
				if string(content) != "original" {
					t.Errorf("File content changed unexpectedly: %q", string(content))
				}
			}

			os.Remove(testFile)
		})
	}
}

func TestWriteFile_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "file.txt")

	inst := &Installer{options: Options{}}
	_, err := inst.writeFile(deepPath, []byte("test"))

	if err != nil {
		t.Fatalf("writeFile() failed to create directories: %v", err)
	}

	if !fileExists(deepPath) {
		t.Errorf("File was not created at deep path")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists() returned false for existing file")
	}
	if fileExists(filepath.Join(tmpDir, "does-not-exist.txt")) {
		t.Error("fileExists() returned true for non-existing file")
	}
}

func TestMatchesFilter(t *testing.T) {
	skill := Skill{
		Name:      "test",
		Tags:      []string{"workflow", "core"},
		Languages: []string{"go", "any"},
	}

	tests := []struct {
		name      string
		tags      []string
		languages []string
		want      bool
	}{
		{"no filter", nil, nil, true},
		{"matching tag", []string{"workflow"}, nil, true},
		{"non-matching tag", []string{"design"}, nil, false},
		{"matching language", nil, []string{"go"}, true},
		{"any language matches all", nil, []string{"python"}, true},
		{"matching tag and language", []string{"core"}, []string{"go"}, true},
		{"non-matching tag with matching language", []string{"design"}, []string{"go"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(skill, tt.tags, tt.languages)
			if got != tt.want {
				t.Errorf("matchesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseYAMLList(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"[a, b, c]", 3},
		{"[single]", 1},
		{"value", 1},
		{"", 0},
		{"[  spaced , items  ]", 2},
	}

	for _, tt := range tests {
		got := parseYAMLList(tt.input)
		if len(got) != tt.want {
			t.Errorf("parseYAMLList(%q) returned %d items, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestGenerateSkillTemplate(t *testing.T) {
	result := GenerateSkillTemplate("my-skill", "A description", "opus", []string{"custom"}, []string{"go"})

	if !strings.Contains(result, "name: my-skill") {
		t.Error("template missing name")
	}
	if !strings.Contains(result, "model: opus") {
		t.Error("template missing model")
	}
	if !strings.Contains(result, "tags: [custom]") {
		t.Error("template missing tags")
	}
	// Verify titleCase is used (not strings.Title)
	if !strings.Contains(result, "# My Skill") {
		t.Error("template missing title-cased heading")
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello world", "Hello World"},
		{"my-skill", "My-skill"},
		{"", ""},
		{"a", "A"},
	}

	for _, tt := range tests {
		got := titleCase(tt.input)
		if got != tt.want {
			t.Errorf("titleCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListSkills(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/a/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: a\ndescription: Skill A\n---\n# A"),
		},
		"skills/b/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: b\ndescription: Skill B\n---\n# B"),
		},
	}

	inst := New(testFS, Options{})

	skills, err := inst.ListSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	// ListAllSkills should return the same
	allSkills, err := inst.ListAllSkills()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(allSkills) != len(skills) {
		t.Errorf("ListAllSkills returned %d, ListSkills returned %d", len(allSkills), len(skills))
	}
}

func TestInstallSkillsForHarness_ClaudeUnchanged(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: my-skill\ndescription: Test\nmodel: sonnet\n---\nRead CLAUDE.md and ~/.claude/skills/my-skill and use TaskCreate."),
		},
		"antigravity/skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: my-skill\ndescription: Antigravity override\n---\n# Antigravity Override"),
		},
		"antigravity/skills/peer-pipeline-swarm/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: peer-pipeline-swarm\ndescription: Swarm\n---\n# Swarm"),
		},
	}

	tmpDir := t.TempDir()
	inst := New(testFS, Options{})
	results, err := inst.InstallSkillsForHarness(tmpDir, HarnessClaude, true, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for claude, got %d: %v", len(results), results)
	}

	got, err := os.ReadFile(filepath.Join(tmpDir, "my-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("failed reading installed skill: %v", err)
	}
	if !strings.Contains(string(got), "CLAUDE.md") || !strings.Contains(string(got), "model: sonnet") {
		t.Errorf("expected claude skill untouched, got:\n%s", string(got))
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "peer-pipeline-swarm", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("did not expect peer-pipeline-swarm in claude install")
	}
}

func TestInstallSkillsForHarness_AntigravityOverlayAndAdaptation(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/domain-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: domain-skill\ndescription: Domain skill\nmodel: opus\neffort: high\ntags: [core]\n---\nCheck ~/.claude/point-of-view and ~/.claude/skills/domain-skill and .claude/skills/other and .claude/agents/rev.md and CLAUDE.md (not CLAUDE-BASE.md or init-claude-md) and TaskCreate.\nKeep line with both GEMINI.md and CLAUDE.md untouched.\nmodel: in-body-stays"),
		},
		"skills/domain-skill/scripts/__pycache__/cached.pyc": &fstest.MapFile{
			Data: []byte("bytecode"),
		},
		"skills/domain-skill/references/config.md": &fstest.MapFile{
			Data: []byte("secret config"),
		},
		"skills/domain-skill/references/team.local.md": &fstest.MapFile{
			Data: []byte("secret team"),
		},
		"skills/codebase-memory-exploring/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: codebase-memory-exploring\ndescription: Local only\n---\n# Local"),
		},
		"skills/overridden-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: overridden-skill\ndescription: Base version\ntags: [workflow]\n---\n# Base"),
		},
		"skills/overridden-skill/scripts/helper.py": &fstest.MapFile{
			Data: []byte("print('helper')\n"),
		},
		"antigravity/skills/overridden-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: overridden-skill\ndescription: Antigravity version\n---\n# Overridden in ~/.gemini/config/skills and ~/.gemini/config/plugins/antigravity-subagents/agents"),
		},
		"antigravity/skills/peer-pipeline-swarm/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: peer-pipeline-swarm\ndescription: Swarm\n---\n# Swarm"),
		},
	}

	agyDir := filepath.Join(t.TempDir(), "agy-skills")
	inst := New(testFS, Options{Force: true})
	_, err := inst.InstallSkillsForHarness(agyDir, HarnessAntigravity, true, nil, nil)
	if err != nil {
		t.Fatalf("InstallSkillsForHarness(antigravity) error: %v", err)
	}

	adaptedBytes, err := os.ReadFile(filepath.Join(agyDir, "domain-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading domain-skill: %v", err)
	}
	adapted := string(adaptedBytes)
	if strings.Contains(adapted, "model: opus") || strings.Contains(adapted, "effort: high") {
		t.Errorf("expected frontmatter model/effort stripped, got:\n%s", adapted)
	}
	if !strings.Contains(adapted, "model: in-body-stays") {
		t.Errorf("expected body model: line preserved, got:\n%s", adapted)
	}
	if !strings.Contains(adapted, "~/.gemini/point-of-view") {
		t.Errorf("expected ~/.gemini/point-of-view rewrite, got:\n%s", adapted)
	}
	if !strings.Contains(adapted, "~/.gemini/config/skills/domain-skill") || !strings.Contains(adapted, "~/.gemini/config/skills/other") {
		t.Errorf("expected antigravity skills path rewrite, got:\n%s", adapted)
	}
	if !strings.Contains(adapted, "~/.gemini/config/plugins/antigravity-subagents/agents/rev.md") {
		t.Errorf("expected antigravity agents path rewrite, got:\n%s", adapted)
	}
	if !strings.Contains(adapted, "and GEMINI.md (not CLAUDE-BASE.md or init-claude-md) and task.md.") {
		t.Errorf("expected word-bounded GEMINI.md and task.md rewrite, got:\n%s", adapted)
	}
	if !strings.Contains(adapted, "Keep line with both GEMINI.md and CLAUDE.md untouched.") {
		t.Errorf("expected dual GEMINI.md/CLAUDE.md line preserved, got:\n%s", adapted)
	}

	overriddenBytes, err := os.ReadFile(filepath.Join(agyDir, "overridden-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading overridden-skill: %v", err)
	}
	if !strings.Contains(string(overriddenBytes), "Antigravity version") {
		t.Errorf("expected antigravity override to win, got:\n%s", string(overriddenBytes))
	}
	if _, err := os.Stat(filepath.Join(agyDir, "overridden-skill", "scripts", "helper.py")); os.IsNotExist(err) {
		t.Errorf("expected non-overridden helper.py from base skill to be preserved")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "peer-pipeline-swarm", "SKILL.md")); os.IsNotExist(err) {
		t.Errorf("expected peer-pipeline-swarm installed for antigravity")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "domain-skill", "scripts", "__pycache__", "cached.pyc")); !os.IsNotExist(err) {
		t.Errorf("expected __pycache__ skipped")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "domain-skill", "references", "config.md")); !os.IsNotExist(err) {
		t.Errorf("expected source references/config.md skipped")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "domain-skill", "references", "team.local.md")); !os.IsNotExist(err) {
		t.Errorf("expected source *.local.md skipped")
	}
	if _, err := os.Stat(filepath.Join(agyDir, "codebase-memory-exploring", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected codebase-memory-* skipped from embedded skills")
	}

	gemDir := filepath.Join(t.TempDir(), "gemini-skills")
	_, err = inst.InstallSkillsForHarness(gemDir, HarnessGemini, true, nil, nil)
	if err != nil {
		t.Fatalf("InstallSkillsForHarness(gemini) error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(gemDir, "peer-pipeline-swarm", "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("expected Antigravity-only peer-pipeline-swarm skipped on gemini target")
	}
	gemOverridden, err := os.ReadFile(filepath.Join(gemDir, "overridden-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading gemini overridden-skill: %v", err)
	}
	if !strings.Contains(string(gemOverridden), "~/.gemini/skills") || !strings.Contains(string(gemOverridden), "~/.gemini/agents") {
		t.Errorf("expected gemini paths rewritten in overlay skill, got:\n%s", string(gemOverridden))
	}
}

func TestInstallSkillsForHarness_PreservesLocalConfigUnderForce(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/team-pulse/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: team-pulse\ndescription: Pulse\n---\n# Pulse"),
		},
	}

	destDir := t.TempDir()
	localPath := filepath.Join(destDir, "team-pulse", "references", "team.local.md")
	configPath := filepath.Join(destDir, "antigravity-orchestration", "references", "config.md")
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, []byte("keep-local"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("keep-config"), 0644); err != nil {
		t.Fatal(err)
	}

	inst := New(testFS, Options{Force: true})
	res1, err := inst.writeFile(localPath, []byte("overwrite-attempt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res1, "SKIP:") {
		t.Errorf("expected SKIP for .local.md under Force, got %q", res1)
	}
	res2, err := inst.writeFile(configPath, []byte("overwrite-attempt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res2, "SKIP:") {
		t.Errorf("expected SKIP for references/config.md under Force, got %q", res2)
	}

	gotLocal, _ := os.ReadFile(localPath)
	gotConfig, _ := os.ReadFile(configPath)
	if string(gotLocal) != "keep-local" || string(gotConfig) != "keep-config" {
		t.Errorf("local files were modified: %q / %q", string(gotLocal), string(gotConfig))
	}
}

func TestUnlinkIfSymlink_RemovesSymlinkWithoutTouchingTarget(t *testing.T) {
	testFS := fstest.MapFS{
		"skills/my-skill/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: my-skill\ndescription: New version\n---\n# New Version"),
		},
	}

	root := t.TempDir()
	repoSkillsDir := filepath.Join(root, "repo-skills")
	if err := os.MkdirAll(filepath.Join(repoSkillsDir, "my-skill"), 0755); err != nil {
		t.Fatal(err)
	}
	sentinelFile := filepath.Join(repoSkillsDir, "my-skill", "SKILL.md")
	if err := os.WriteFile(sentinelFile, []byte("ORIGINAL REPO CONTENT"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkDest := filepath.Join(root, "linked-skills")
	if err := os.Symlink(repoSkillsDir, symlinkDest); err != nil {
		t.Fatal(err)
	}

	dryInst := New(testFS, Options{DryRun: true})
	dryRes, err := dryInst.InstallSkillsForHarness(symlinkDest, HarnessAntigravity, true, nil, nil)
	if err != nil {
		t.Fatalf("dry-run InstallSkillsForHarness error: %v", err)
	}
	info, err := os.Lstat(symlinkDest)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink preserved on dry-run")
	}
	joinedDry := strings.Join(dryRes, "\n")
	if !strings.Contains(joinedDry, "WOULD UNLINK SYMLINK:") || !strings.Contains(joinedDry, "WOULD CREATE:") {
		t.Errorf("expected WOULD UNLINK SYMLINK and WOULD CREATE in dry-run output, got:\n%s", joinedDry)
	}

	realInst := New(testFS, Options{Force: true})
	_, err = realInst.InstallSkillsForHarness(symlinkDest, HarnessAntigravity, true, nil, nil)
	if err != nil {
		t.Fatalf("InstallSkillsForHarness error: %v", err)
	}

	info, err = os.Lstat(symlinkDest)
	if err != nil {
		t.Fatalf("stat symlinkDest: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		t.Errorf("expected symlinkDest replaced with real directory, mode=%v", info.Mode())
	}

	sentinelBytes, err := os.ReadFile(sentinelFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(sentinelBytes) != "ORIGINAL REPO CONTENT" {
		t.Errorf("repo sentinel file was overwritten through symlink! got: %q", string(sentinelBytes))
	}
}

func TestInstallAgentsForHarness_AntigravityAndGemini(t *testing.T) {
	testFS := fstest.MapFS{
		"agents/debugger.md": &fstest.MapFile{
			Data: []byte("---\nname: debugger\nmodel: sonnet\n---\n# Claude Debugger"),
		},
		"antigravity/plugins/antigravity-subagents/plugin.json": &fstest.MapFile{
			Data: []byte(`{"name":"antigravity-subagents","version":"1.2.0"}`),
		},
		"antigravity/plugins/antigravity-subagents/agents/implementer.md": &fstest.MapFile{
			Data: []byte("---\nname: implementer\n---\nSee ~/.gemini/config/skills"),
		},
	}

	root := t.TempDir()
	agyAgentsDir := filepath.Join(root, "plugins", "antigravity-subagents", "agents")
	inst := New(testFS, Options{Force: true})

	_, err := inst.InstallAgentsForHarness(agyAgentsDir, HarnessAntigravity, true, nil)
	if err != nil {
		t.Fatalf("InstallAgentsForHarness(antigravity) error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agyAgentsDir, "implementer.md")); os.IsNotExist(err) {
		t.Errorf("expected implementer.md installed for antigravity")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(agyAgentsDir), "plugin.json")); os.IsNotExist(err) {
		t.Errorf("expected plugin.json installed for global antigravity agents")
	}

	gemAgentsDir := filepath.Join(root, "gemini-agents")
	_, err = inst.InstallAgentsForHarness(gemAgentsDir, HarnessGemini, true, nil)
	if err != nil {
		t.Fatalf("InstallAgentsForHarness(gemini) error: %v", err)
	}
	gemImpl, err := os.ReadFile(filepath.Join(gemAgentsDir, "implementer.md"))
	if err != nil {
		t.Fatalf("expected implementer.md for gemini: %v", err)
	}
	if !strings.Contains(string(gemImpl), "~/.gemini/skills") {
		t.Errorf("expected ~/.gemini/skills rewrite for gemini agent, got: %s", string(gemImpl))
	}
}

func TestInstallGlobalHarnessAssets_MergesHooksJSONPreservingExtraKeys(t *testing.T) {
	testFS := fstest.MapFS{
		"antigravity/GEMINI.md": &fstest.MapFile{
			Data: []byte("# Global GEMINI Rules\nPath: ~/.gemini/config/skills"),
		},
		"antigravity/hooks.json": &fstest.MapFile{
			Data: []byte(`{"autonomous-circuit-breaker":{"hooks":{"PreToolUse":[{"matcher":"swarm"}]}}}`),
		},
		"antigravity/hooks/autonomous-breaker.sh": &fstest.MapFile{
			Data: []byte("#!/usr/bin/env bash\nexit 0\n"),
		},
		"antigravity/hooks/autonomous_breaker.py": &fstest.MapFile{
			Data: []byte("print('ok')\n"),
		},
		"antigravity/hooks/test_autonomous_breaker.py": &fstest.MapFile{
			Data: []byte("def test_noop(): pass\n"),
		},
	}

	geminiHome := t.TempDir()
	configDir := filepath.Join(geminiHome, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	hooksJSONPath := filepath.Join(configDir, "hooks.json")
	existingJSON := `{"impeccable-design-detector":{"enabled":true},"autonomous-circuit-breaker":{"old":true}}`
	if err := os.WriteFile(hooksJSONPath, []byte(existingJSON), 0644); err != nil {
		t.Fatal(err)
	}

	inst := New(testFS, Options{Force: true})
	_, err := inst.InstallGlobalHarnessAssets(geminiHome, HarnessAntigravity)
	if err != nil {
		t.Fatalf("InstallGlobalHarnessAssets error: %v", err)
	}

	mergedBytes, err := os.ReadFile(hooksJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	merged := string(mergedBytes)
	if !strings.Contains(merged, "impeccable-design-detector") {
		t.Errorf("expected impeccable-design-detector preserved in hooks.json, got:\n%s", merged)
	}
	if !strings.Contains(merged, `"matcher": "swarm"`) {
		t.Errorf("expected autonomous-circuit-breaker updated in hooks.json, got:\n%s", merged)
	}

	shInfo, err := os.Stat(filepath.Join(configDir, "hooks", "autonomous-breaker.sh"))
	if err != nil {
		t.Fatalf("autonomous-breaker.sh not installed: %v", err)
	}
	if shInfo.Mode().Perm()&0111 == 0 {
		t.Errorf("expected autonomous-breaker.sh to be executable, mode=%v", shInfo.Mode())
	}
	if _, err := os.Stat(filepath.Join(configDir, "hooks", "test_autonomous_breaker.py")); !os.IsNotExist(err) {
		t.Errorf("expected test_autonomous_breaker.py excluded from installed hooks")
	}

	noForceInst := New(testFS, Options{Force: false})
	skipRes, err := noForceInst.InstallGlobalHarnessAssets(geminiHome, HarnessAntigravity)
	if err != nil {
		t.Fatalf("noForce InstallGlobalHarnessAssets error: %v", err)
	}
	if !strings.Contains(strings.Join(skipRes, "\n"), "SKIP: "+hooksJSONPath) {
		t.Errorf("expected SKIP on hooks.json without Force, got:\n%v", skipRes)
	}

	dryInst := New(testFS, Options{Force: true, DryRun: true})
	dryRes, err := dryInst.InstallGlobalHarnessAssets(geminiHome, HarnessAntigravity)
	if err != nil {
		t.Fatalf("dryRun InstallGlobalHarnessAssets error: %v", err)
	}
	if !strings.Contains(strings.Join(dryRes, "\n"), "WOULD OVERWRITE: "+hooksJSONPath) {
		t.Errorf("expected WOULD OVERWRITE on hooks.json dry-run with Force, got:\n%v", dryRes)
	}

	if err := os.WriteFile(hooksJSONPath, []byte("{broken json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := inst.InstallGlobalHarnessAssets(geminiHome, HarnessAntigravity); err == nil {
		t.Errorf("expected error when existing hooks.json has malformed JSON")
	}
}

func TestInstallGlobalHarnessAssets_DotfilesSymlinkAndSymlinkedHooksAndGeminiMD(t *testing.T) {
	testFS := fstest.MapFS{
		"antigravity/GEMINI.md":            &fstest.MapFile{Data: []byte("global rules ~/.gemini/config/skills/antigravity-orchestration/SKILL.md\n")},
		"antigravity/hooks.json":           &fstest.MapFile{Data: []byte(`{"autonomous-circuit-breaker":{"enabled":true,"PreToolUse":[{"matcher":"read_file"}]}}`)},
		"antigravity/rules/org.example.md": &fstest.MapFile{Data: []byte("# Org Example\n")},
	}

	tmp := t.TempDir()
	realDotfiles := filepath.Join(tmp, "dotfiles-gemini")
	if err := os.MkdirAll(realDotfiles, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDotfiles, "oauth_creds.json"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	symlinkedHome := filepath.Join(tmp, ".gemini")
	if err := os.Symlink(realDotfiles, symlinkedHome); err != nil {
		t.Fatal(err)
	}

	inst := New(testFS, Options{Force: true})
	if _, err := inst.InstallGlobalHarnessAssets(symlinkedHome, HarnessAntigravity); err == nil {
		t.Fatalf("expected error when geminiHome itself is a symlink")
	}
	if info, err := os.Lstat(symlinkedHome); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlinkedHome was unlinked despite being root geminiHome")
	}

	realGeminiHome := filepath.Join(tmp, "real-gemini")
	if err := os.MkdirAll(filepath.Join(realGeminiHome, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	externalHooks := filepath.Join(tmp, "external-hooks.json")
	if err := os.WriteFile(externalHooks, []byte(`{"impeccable-design-detector":{"enabled":true,"PreToolUse":[{"matcher":"edit_file"}]}}`), 0644); err != nil {
		t.Fatal(err)
	}
	hooksDest := filepath.Join(realGeminiHome, "config", "hooks.json")
	if err := os.Symlink(externalHooks, hooksDest); err != nil {
		t.Fatal(err)
	}

	if _, err := inst.InstallGlobalHarnessAssets(realGeminiHome, HarnessAntigravity); err != nil {
		t.Fatalf("InstallGlobalHarnessAssets(Antigravity) failed: %v", err)
	}
	mergedHooks, err := os.ReadFile(hooksDest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mergedHooks), "impeccable-design-detector") {
		t.Errorf("symlinked hooks.json lost impeccable-design-detector after merge:\n%s", string(mergedHooks))
	}
	if _, err := os.Stat(filepath.Join(realGeminiHome, "config", "rules", "org.example.md")); err != nil {
		t.Errorf("expected config/rules/org.example.md to be installed: %v", err)
	}

	if _, err := inst.InstallGlobalHarnessAssets(realGeminiHome, HarnessGemini); err != nil {
		t.Fatalf("InstallGlobalHarnessAssets(Gemini) failed: %v", err)
	}
	geminiMDBytes, err := os.ReadFile(filepath.Join(realGeminiHome, "GEMINI.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(geminiMDBytes), "~/.gemini/config/skills/antigravity-orchestration/SKILL.md") {
		t.Errorf("HarnessGemini overwrote shared ~/.gemini/GEMINI.md:\n%s", string(geminiMDBytes))
	}
}

func TestInstallFromLocal_AndWriteFile_SymlinkExfiltrationAndPreservedLocalSymlink(t *testing.T) {
	tmp := t.TempDir()
	secretFile := filepath.Join(tmp, "host-secret.txt")
	if err := os.WriteFile(secretFile, []byte("TOP_SECRET_SSH_KEY"), 0600); err != nil {
		t.Fatal(err)
	}

	srcDir := filepath.Join(tmp, "src-skills", "demo")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("---\nname: demo\ndescription: ok\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secretFile, filepath.Join(srcDir, "leaked.md")); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(tmp, "dest-skills")
	inst := New(fstest.MapFS{}, Options{Force: true})
	if _, err := inst.InstallFromLocal(filepath.Join(tmp, "src-skills"), destDir); err != nil {
		t.Fatalf("InstallFromLocal error: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(destDir, "demo", "leaked.md")); !os.IsNotExist(err) {
		t.Errorf("expected symlinked source file leaked.md to be skipped, but it exists in destDir")
	}

	localSecret := filepath.Join(tmp, "my-team.local.md")
	if err := os.WriteFile(localSecret, []byte("REAL_TEAM_DATA"), 0644); err != nil {
		t.Fatal(err)
	}
	preservedSymlink := filepath.Join(destDir, "demo", "references", "team.local.md")
	if err := os.MkdirAll(filepath.Dir(preservedSymlink), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(localSecret, preservedSymlink); err != nil {
		t.Fatal(err)
	}
	if _, err := inst.writeFileWithRoot(destDir, preservedSymlink, []byte("OVERWRITTEN"), 0644); err != nil {
		t.Fatalf("writeFileWithRoot on preserved symlink error: %v", err)
	}
	info, err := os.Lstat(preservedSymlink)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected preserved *.local.md symlink to remain intact")
	}
	gotLocal, _ := os.ReadFile(localSecret)
	if string(gotLocal) != "REAL_TEAM_DATA" {
		t.Errorf("preserved local symlink target was modified: %s", string(gotLocal))
	}
}

func TestParseSkill_FoldedDescription(t *testing.T) {
	content := []byte("---\nname: folded-skill\ndescription: >-\n  First line of description\n  second line of description.\ntags: [workflow]\n---\n# Folded")
	skill, err := parseSkill(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "First line of description second line of description."
	if skill.Description != want {
		t.Errorf("got description %q, want %q", skill.Description, want)
	}
}

func TestInstallSkills_SkipsPythonTestFiles(t *testing.T) {
	testFS := fstest.MapFS{
		"shared/skills/demo/SKILL.md": &fstest.MapFile{
			Data: []byte("---\nname: demo\ndescription: Demo\n---\n# Demo"),
		},
		"shared/skills/demo/scripts/check_demo.py": &fstest.MapFile{
			Data: []byte("print('ok')\n"),
		},
		"shared/skills/demo/scripts/test_check_demo.py": &fstest.MapFile{
			Data: []byte("def test_ok(): pass\n"),
		},
		"shared/skills/demo/scripts/check_demo_test.py": &fstest.MapFile{
			Data: []byte("def test_ok2(): pass\n"),
		},
	}

	destDir := t.TempDir()
	inst := New(testFS, Options{Force: true})
	if _, err := inst.InstallSkills(destDir, nil, nil); err != nil {
		t.Fatalf("InstallSkills error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "demo", "scripts", "check_demo.py")); err != nil {
		t.Errorf("expected check_demo.py to be installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "demo", "scripts", "test_check_demo.py")); !os.IsNotExist(err) {
		t.Errorf("expected test_check_demo.py to be skipped during install")
	}
	if _, err := os.Stat(filepath.Join(destDir, "demo", "scripts", "check_demo_test.py")); !os.IsNotExist(err) {
		t.Errorf("expected check_demo_test.py to be skipped during install")
	}
}
