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
