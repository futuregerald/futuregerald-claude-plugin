package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDetectGoProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goMod := `module github.com/user/myapp

go 1.21
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectGoProject(dir)

	if info.Name != "myapp" {
		t.Errorf("Name = %q, want %q", info.Name, "myapp")
	}
	if info.Framework != "Go" {
		t.Errorf("Framework = %q, want %q", info.Framework, "Go")
	}
	if info.TestCommand != "go test ./..." {
		t.Errorf("TestCommand = %q, want %q", info.TestCommand, "go test ./...")
	}
	if info.LanguageTemplate != "go.md" {
		t.Errorf("LanguageTemplate = %q, want %q", info.LanguageTemplate, "go.md")
	}
	if info.BuildCommand != "go build ./..." {
		t.Errorf("BuildCommand = %q, want %q", info.BuildCommand, "go build ./...")
	}
}

func TestDetectNodeProject_React(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkg := `{
  "name": "my-react-app",
  "dependencies": {
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectNodeProject(dir)

	if info.Framework != "React" {
		t.Errorf("Framework = %q, want %q", info.Framework, "React")
	}
	if info.LanguageTemplate != "react.md" {
		t.Errorf("LanguageTemplate = %q, want %q", info.LanguageTemplate, "react.md")
	}
}

func TestDetectNodeProject_NextJS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkg := `{
  "name": "my-next-app",
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectNodeProject(dir)

	if info.Framework != "Next.js" {
		t.Errorf("Framework = %q, want %q (should prefer Next.js over React)", info.Framework, "Next.js")
	}
	if info.LanguageTemplate != "react.md" {
		t.Errorf("LanguageTemplate = %q, want %q", info.LanguageTemplate, "react.md")
	}
}

func TestDetectNodeProject_AdonisJS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkg := `{
  "name": "my-adonis-app",
  "dependencies": {
    "@adonisjs/core": "^6.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectNodeProject(dir)

	if info.Framework != "AdonisJS" {
		t.Errorf("Framework = %q, want %q", info.Framework, "AdonisJS")
	}
	if info.LanguageTemplate != "adonisjs.md" {
		t.Errorf("LanguageTemplate = %q, want %q", info.LanguageTemplate, "adonisjs.md")
	}
}

func TestDetectNodeProject_Express(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkg := `{
  "name": "my-express-app",
  "dependencies": {
    "express": "^4.18.0"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectNodeProject(dir)

	if info.Framework != "Express" {
		t.Errorf("Framework = %q, want %q", info.Framework, "Express")
	}
	if info.LanguageTemplate != "nodejs.md" {
		t.Errorf("LanguageTemplate = %q, want %q", info.LanguageTemplate, "nodejs.md")
	}
}

func TestDetectNodeProject_ScriptOverride(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkg := `{
  "name": "my-app",
  "dependencies": {
    "react": "^18.0.0"
  },
  "scripts": {
    "typecheck": "tsc --noEmit",
    "test": "vitest",
    "build": "vite build"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectNodeProject(dir)

	if info.TypecheckCommand != "npm run typecheck" {
		t.Errorf("TypecheckCommand = %q, want %q", info.TypecheckCommand, "npm run typecheck")
	}
	if info.TestCommand != "npm test" {
		t.Errorf("TestCommand = %q, want %q", info.TestCommand, "npm test")
	}
	if info.BuildCommand != "npm run build" {
		t.Errorf("BuildCommand = %q, want %q", info.BuildCommand, "npm run build")
	}
}

func TestDetectRustProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cargo := `[package]
name = "my-rust-tool"
description = "A blazingly fast CLI tool"
version = "0.1.0"
edition = "2021"

[dependencies]
`
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectRustProject(dir)

	if info.Name != "my-rust-tool" {
		t.Errorf("Name = %q, want %q", info.Name, "my-rust-tool")
	}
	if info.Description != "A blazingly fast CLI tool" {
		t.Errorf("Description = %q, want %q", info.Description, "A blazingly fast CLI tool")
	}
	if info.Framework != "Rust" {
		t.Errorf("Framework = %q, want %q", info.Framework, "Rust")
	}
	if info.TestCommand != "cargo test" {
		t.Errorf("TestCommand = %q, want %q", info.TestCommand, "cargo test")
	}
}

func TestDetectPythonProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pyproject := `[project]
name = "my-python-lib"
version = "1.0.0"
description = "A Python library"

[build-system]
requires = ["setuptools"]
`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatal(err)
	}

	info := detectPythonProject(dir)

	if info.Name != "my-python-lib" {
		t.Errorf("Name = %q, want %q", info.Name, "my-python-lib")
	}
	if info.TestCommand != "pytest" {
		t.Errorf("TestCommand = %q, want %q", info.TestCommand, "pytest")
	}
	if info.Framework != "Python" {
		t.Errorf("Framework = %q, want %q", info.Framework, "Python")
	}
}

func TestDetectKeyDirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create subdirectories
	for _, name := range []string{"src", "tests", "docs"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
			t.Fatal(err)
		}
	}

	found := detectKeyDirectories(dir)

	expected := map[string]bool{
		"src/":   false,
		"tests/": false,
		"docs/":  false,
	}
	for _, d := range found {
		if _, ok := expected[d]; ok {
			expected[d] = true
		}
	}
	for d, seen := range expected {
		if !seen {
			t.Errorf("expected directory %q not found in result %v", d, found)
		}
	}
}

func TestDetectProject_Fallback(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	info := detectProject(dir)

	// Name should fall back to the directory basename
	basename := filepath.Base(dir)
	if info.Name != basename {
		t.Errorf("Name = %q, want %q (dir basename)", info.Name, basename)
	}
	if info.Framework != "" {
		t.Errorf("Framework = %q, want empty string", info.Framework)
	}
}

func TestRemoveUnfilledPlaceholders(t *testing.T) {
	t.Parallel()

	input := "Hello {{FOO}} world {{BAR_BAZ}}"
	result := placeholderRe.ReplaceAllString(input, "")

	if strings.Contains(result, "{{FOO}}") {
		t.Errorf("result still contains {{FOO}}: %q", result)
	}
	if strings.Contains(result, "{{BAR_BAZ}}") {
		t.Errorf("result still contains {{BAR_BAZ}}: %q", result)
	}
	expected := "Hello  world "
	if result != expected {
		t.Errorf("result = %q, want %q", result, expected)
	}
}

func TestRemoveEmptySections(t *testing.T) {
	t.Parallel()

	input := "## Project Overview\n\n---"
	result := removeEmptySections(input)

	if strings.Contains(result, "## Project Overview") {
		t.Errorf("empty section was not removed: %q", result)
	}
}

func TestRemoveEmptySections_EmptyCodeFence(t *testing.T) {
	t.Parallel()

	input := "## Quick Reference\n\n```bash\n```\n\n---"
	result := removeEmptySections(input)

	if strings.Contains(result, "## Quick Reference") {
		t.Errorf("section with empty code fence was not removed: %q", result)
	}
}

func TestApplyProjectDetection_EmptyTypecheck(t *testing.T) {
	t.Parallel()

	// Template mimics the real CLAUDE-BASE.md layout with table and code fence
	baseTemplate := "# {{PROJECT_NAME}}\n\n" +
		"| 5. TEST | `{{TEST_COMMAND}}` + `{{TYPECHECK_COMMAND}}` | — | Zero failures |\n\n" +
		"## Quick Reference\n\n" +
		"```bash\n{{TEST_COMMAND}}\n{{TYPECHECK_COMMAND}}\n```\n\n---\n"

	mockFS := fstest.MapFS{}

	info := ProjectInfo{
		Name:        "mygoapp",
		Framework:   "Go",
		TestCommand: "go test ./...",
		// TypecheckCommand intentionally empty
	}

	result := string(applyProjectDetection([]byte(baseTemplate), info, mockFS))

	// Table row should NOT contain empty backtick artifact ("` + ` `")
	if strings.Contains(result, "+ ` `") {
		t.Errorf("result still contains empty backtick artifact: %q", result)
	}
	// Table should still have test command in backticks
	if !strings.Contains(result, "`go test ./...`") {
		t.Errorf("test command backticks missing from table")
	}

	// Quick Reference code fence should NOT have blank lines
	if strings.Contains(result, "```bash\n\n") {
		t.Errorf("code fence has leading blank line")
	}
	if strings.Contains(result, "\n\n```") {
		t.Errorf("code fence has trailing blank line")
	}
	// Test command should be present in the code fence
	if !strings.Contains(result, "```bash\ngo test ./...") {
		t.Errorf("test command not found in code fence: %q", result)
	}
}

func TestApplyProjectDetection(t *testing.T) {
	t.Parallel()

	baseTemplate := `# {{PROJECT_NAME}} - Claude Code Configuration

## Project Overview

{{PROJECT_DESCRIPTION}}

## Key Directories

{{KEY_DIRECTORIES}}

---

## Quick Reference

` + "```bash" + `
{{TEST_COMMAND}}
{{TYPECHECK_COMMAND}}
` + "```" + `

---

<!-- LANGUAGE_SPECIFIC -->
`

	goLangTemplate := `## Go Rules

- Follow gofmt conventions
- Use table-driven tests
`

	mockFS := fstest.MapFS{
		"templates/languages/go.md": &fstest.MapFile{
			Data: []byte(goLangTemplate),
		},
	}

	info := ProjectInfo{
		Name:             "myapp",
		Description:      "A great application",
		Framework:        "Go",
		LanguageTemplate: "go.md",
		TestCommand:      "go test ./...",
		TypecheckCommand: "go vet ./...",
		BuildCommand:     "go build ./...",
		KeyDirectories:   []string{"src/", "tests/"},
	}

	result := string(applyProjectDetection([]byte(baseTemplate), info, mockFS))

	// All named placeholders should be replaced
	if strings.Contains(result, "{{") {
		t.Errorf("result still contains unfilled placeholders: %q", result)
	}

	// Project name should appear in the title
	if !strings.Contains(result, "# myapp - Claude Code Configuration") {
		t.Errorf("project name not replaced in title")
	}

	// Description should be present
	if !strings.Contains(result, "A great application") {
		t.Errorf("description not replaced")
	}

	// Key directories should be formatted
	if !strings.Contains(result, "- `src/`") {
		t.Errorf("key directory src/ not found")
	}
	if !strings.Contains(result, "- `tests/`") {
		t.Errorf("key directory tests/ not found")
	}

	// Test command should appear
	if !strings.Contains(result, "go test ./...") {
		t.Errorf("test command not found in result")
	}

	// Language template should be inserted
	if !strings.Contains(result, "## Go Rules") {
		t.Errorf("language template not inserted")
	}
	if !strings.Contains(result, "table-driven tests") {
		t.Errorf("language template content not found")
	}

	// LANGUAGE_SPECIFIC marker should be gone
	if strings.Contains(result, "<!-- LANGUAGE_SPECIFIC -->") {
		t.Errorf("LANGUAGE_SPECIFIC marker still present")
	}
}
