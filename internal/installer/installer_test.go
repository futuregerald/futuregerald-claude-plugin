package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseSkill(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantName    string
		wantDesc    string
		wantModel   string
		wantErr     bool
	}{
		{
			name: "valid skill with all fields",
			content: `---
name: test-skill
description: A test skill for testing
model: haiku
---

# Test Skill

Content here.`,
			wantName:  "test-skill",
			wantDesc:  "A test skill for testing",
			wantModel: "haiku",
			wantErr:   false,
		},
		{
			name: "skill without model",
			content: `---
name: no-model-skill
description: Skill without model specified
---

# No Model Skill`,
			wantName:  "no-model-skill",
			wantDesc:  "Skill without model specified",
			wantModel: "",
			wantErr:   false,
		},
		{
			name: "skill missing name",
			content: `---
description: Missing name
model: sonnet
---

# Missing Name`,
			wantErr: true,
		},
		{
			name:    "no frontmatter",
			content: "# Just content\n\nNo frontmatter here.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill, err := parseSkill([]byte(tt.content))

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSkill() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseSkill() unexpected error: %v", err)
				return
			}

			if skill.Name != tt.wantName {
				t.Errorf("parseSkill() name = %q, want %q", skill.Name, tt.wantName)
			}
			if skill.Description != tt.wantDesc {
				t.Errorf("parseSkill() description = %q, want %q", skill.Description, tt.wantDesc)
			}
			if skill.Model != tt.wantModel {
				t.Errorf("parseSkill() model = %q, want %q", skill.Model, tt.wantModel)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing file",
			path: existingFile,
			want: true,
		},
		{
			name: "non-existing file",
			path: filepath.Join(tmpDir, "does-not-exist.txt"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileExists(tt.path); got != tt.want {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestInstaller_WriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		options    Options
		setupFile  bool // Create file before test
		wantPrefix string
		wantWrite  bool // Should file be written
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

			// Verify file state
			content, err := os.ReadFile(testFile)
			if tt.wantWrite {
				if err != nil {
					t.Errorf("Expected file to exist after write")
				} else if string(content) != "new content" {
					t.Errorf("File content = %q, want %q", string(content), "new content")
				}
			} else if tt.setupFile {
				// File should have original content if not written
				if string(content) != "original" {
					t.Errorf("File content changed unexpectedly: %q", string(content))
				}
			}

			// Cleanup for next iteration
			os.Remove(testFile)
		})
	}
}

func TestInstaller_CreateDirectory(t *testing.T) {
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

// Integration test using the real embedded filesystem
// This test is skipped when running in isolation without embedded content
func TestIntegration_InstallSkills(t *testing.T) {
	// Create a test embedded filesystem
	testFS := fstest.MapFS{
		"skills/test-skill.md": &fstest.MapFile{
			Data: []byte(`---
name: test-skill
description: A test skill
model: haiku
---

# Test Skill
`),
		},
		"templates/agents.md": &fstest.MapFile{
			Data: []byte(`# AI Agent Guidelines

Test agents.md content.
`),
		},
	}

	tmpDir := t.TempDir()

	// We can't directly use fstest.MapFS with embed.FS
	// So we'll test the individual functions instead
	t.Run("parseSkill", func(t *testing.T) {
		content := testFS["skills/test-skill.md"].Data
		skill, err := parseSkill(content)
		if err != nil {
			t.Fatalf("parseSkill() error: %v", err)
		}
		if skill.Name != "test-skill" {
			t.Errorf("skill.Name = %q, want %q", skill.Name, "test-skill")
		}
	})

	t.Run("writeFile creates skill", func(t *testing.T) {
		skillPath := filepath.Join(tmpDir, ".claude", "skills", "test.md")
		inst := &Installer{options: Options{}}

		result, err := inst.writeFile(skillPath, testFS["skills/test-skill.md"].Data)
		if err != nil {
			t.Fatalf("writeFile() error: %v", err)
		}

		if !strings.HasPrefix(result, "CREATED:") {
			t.Errorf("Expected CREATED result, got: %s", result)
		}

		if !fileExists(skillPath) {
			t.Error("Skill file was not created")
		}
	})

	t.Run("writeFile creates agents.md", func(t *testing.T) {
		agentsPath := filepath.Join(tmpDir, "agents.md")
		inst := &Installer{options: Options{}}

		result, err := inst.writeFile(agentsPath, testFS["templates/agents.md"].Data)
		if err != nil {
			t.Fatalf("writeFile() error: %v", err)
		}

		if !strings.HasPrefix(result, "CREATED:") {
			t.Errorf("Expected CREATED result, got: %s", result)
		}

		content, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatalf("Failed to read agents.md: %v", err)
		}

		if !strings.Contains(string(content), "AI Agent Guidelines") {
			t.Error("agents.md missing expected content")
		}
	})
}
