package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectInfo holds detected project metadata for template placeholder replacement.
type ProjectInfo struct {
	Name             string
	Description      string
	Framework        string
	LanguageTemplate string
	TestCommand      string
	TypecheckCommand string
	BuildCommand     string
	KeyDirectories   []string
}

// packageJSON is a minimal struct for parsing package.json fields.
type packageJSON struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func mergeMaps(a, b map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

func hasKeyPrefix(m map[string]string, prefix string) bool {
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

// extractTOMLString extracts a string value from a simple TOML `key = "value"` line.
// Only handles double-quoted values on a single line. Sufficient for name/description fields.
func extractTOMLString(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(parts[1]), "\"")
}

func formatKeyDirectories(dirs []string) string {
	if len(dirs) == 0 {
		return ""
	}
	var lines []string
	for _, d := range dirs {
		lines = append(lines, fmt.Sprintf("- `%s`", d))
	}
	return strings.Join(lines, "\n")
}

// placeholderRe matches {{PLACEHOLDER}} patterns in templates.
var placeholderRe = regexp.MustCompile(`\{\{[A-Z_]+\}\}`)

func detectKeyDirectories(dir string) []string {
	candidates := []string{
		"src", "lib", "app", "cmd", "internal", "pkg",
		"api", "server", "client", "web", "frontend", "backend",
		"tests", "test", "spec", "e2e",
		"docs", "config", "scripts", "migrations", "public",
		"resources", "inertia",
	}
	var found []string
	for _, name := range candidates {
		info, err := os.Stat(filepath.Join(dir, name))
		if err == nil && info.IsDir() {
			found = append(found, name+"/")
		}
	}
	return found
}

func detectGoProject(dir string) ProjectInfo {
	info := ProjectInfo{
		Framework:        "Go",
		LanguageTemplate: "go.md",
		TestCommand:      "go test ./...",
		BuildCommand:     "go build ./...",
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return info
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			parts := strings.Split(modulePath, "/")
			info.Name = parts[len(parts)-1]
			break
		}
	}
	return info
}

func detectRustProject(dir string) ProjectInfo {
	info := ProjectInfo{
		Framework:        "Rust",
		LanguageTemplate: "rust.md",
		TestCommand:      "cargo test",
		BuildCommand:     "cargo build",
	}
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return info
	}
	inPackage := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[package]" {
			inPackage = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inPackage = false
			continue
		}
		if !inPackage {
			continue
		}
		if strings.HasPrefix(trimmed, "name") {
			info.Name = extractTOMLString(trimmed)
		} else if strings.HasPrefix(trimmed, "description") {
			info.Description = extractTOMLString(trimmed)
		}
	}
	return info
}

func detectPythonProject(dir string) ProjectInfo {
	info := ProjectInfo{
		Framework:        "Python",
		LanguageTemplate: "python.md",
		TestCommand:      "pytest",
		TypecheckCommand: "mypy .",
	}
	data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	if err != nil {
		return info
	}
	inProject := false
	inPoetry := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[project]" {
			inProject = true
			inPoetry = false
			continue
		}
		if trimmed == "[tool.poetry]" {
			inPoetry = true
			inProject = false
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inProject = false
			inPoetry = false
			continue
		}
		if inProject || inPoetry {
			if strings.HasPrefix(trimmed, "name") {
				info.Name = extractTOMLString(trimmed)
			} else if strings.HasPrefix(trimmed, "description") {
				info.Description = extractTOMLString(trimmed)
			}
		}
	}
	return info
}

func detectRubyProject(dir string) ProjectInfo {
	info := ProjectInfo{
		Framework:        "Ruby",
		LanguageTemplate: "ruby.md",
		TestCommand:      "bundle exec rspec",
		BuildCommand:     "bundle exec rake build",
	}
	data, err := os.ReadFile(filepath.Join(dir, "Gemfile"))
	if err != nil {
		return info
	}
	content := string(data)
	if strings.Contains(content, "'rails'") || strings.Contains(content, "\"rails\"") {
		info.Framework = "Rails"
		info.TestCommand = "bundle exec rspec"
		info.BuildCommand = "bundle exec rails assets:precompile"
	}
	return info
}

func detectPHPProject(dir string) ProjectInfo {
	info := ProjectInfo{
		Framework:        "PHP",
		LanguageTemplate: "php.md",
		TestCommand:      "vendor/bin/phpunit",
		TypecheckCommand: "vendor/bin/phpstan analyse",
		BuildCommand:     "composer install",
	}
	data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err != nil {
		return info
	}
	var composer struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Require     map[string]string `json:"require"`
	}
	if err := json.Unmarshal(data, &composer); err != nil {
		return info
	}
	info.Name = composer.Name
	info.Description = composer.Description
	if _, ok := composer.Require["laravel/framework"]; ok {
		info.Framework = "Laravel"
		info.TestCommand = "php artisan test"
	}
	return info
}

func detectNodeProject(dir string) ProjectInfo {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ProjectInfo{}
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ProjectInfo{}
	}

	info := ProjectInfo{
		Name:        pkg.Name,
		Description: pkg.Description,
	}

	allDeps := mergeMaps(pkg.Dependencies, pkg.DevDependencies)

	// Priority order: most specific first. Next.js MUST come before React
	// because Next.js projects always have react as a dependency too.
	if hasKeyPrefix(allDeps, "@adonisjs/") {
		info.Framework = "AdonisJS"
		info.LanguageTemplate = "adonisjs.md"
		info.TestCommand = "node ace test"
		info.TypecheckCommand = "npx tsc --noEmit"
		info.BuildCommand = "node ace build"
	} else if _, ok := allDeps["svelte"]; ok {
		info.Framework = "Svelte"
		info.LanguageTemplate = "svelte.md"
		info.TestCommand = "npm test"
		info.TypecheckCommand = "npx svelte-check"
		info.BuildCommand = "npm run build"
	} else if _, ok := allDeps["next"]; ok {
		info.Framework = "Next.js"
		info.LanguageTemplate = "react.md"
		info.TestCommand = "npm test"
		info.TypecheckCommand = "npx tsc --noEmit"
		info.BuildCommand = "npm run build"
	} else if _, ok := allDeps["react"]; ok {
		info.Framework = "React"
		info.LanguageTemplate = "react.md"
		info.TestCommand = "npm test"
		info.TypecheckCommand = "npx tsc --noEmit"
		info.BuildCommand = "npm run build"
	} else if _, ok := allDeps["express"]; ok {
		info.Framework = "Express"
		info.LanguageTemplate = "nodejs.md"
		info.TestCommand = "npm test"
		info.TypecheckCommand = "npx tsc --noEmit"
		info.BuildCommand = "npm run build"
	} else {
		info.Framework = "Node.js"
		info.LanguageTemplate = "nodejs.md"
		info.TestCommand = "npm test"
		info.TypecheckCommand = "npx tsc --noEmit"
		info.BuildCommand = "npm run build"
	}

	return overrideFromScripts(info, pkg.Scripts)
}

func overrideFromScripts(info ProjectInfo, scripts map[string]string) ProjectInfo {
	if scripts == nil {
		return info
	}
	if _, ok := scripts["test"]; ok {
		info.TestCommand = "npm test"
	} else if _, ok := scripts["test:unit"]; ok {
		info.TestCommand = "npm run test:unit"
	}
	if _, ok := scripts["typecheck"]; ok {
		info.TypecheckCommand = "npm run typecheck"
	} else if _, ok := scripts["type-check"]; ok {
		info.TypecheckCommand = "npm run type-check"
	} else if _, ok := scripts["check-types"]; ok {
		info.TypecheckCommand = "npm run check-types"
	}
	if _, ok := scripts["build"]; ok {
		info.BuildCommand = "npm run build"
	}
	return info
}

func detectProject(dir string) ProjectInfo {
	info := ProjectInfo{}
	pjPath := filepath.Join(dir, "package.json")
	goModPath := filepath.Join(dir, "go.mod")
	cargoPath := filepath.Join(dir, "Cargo.toml")
	reqsPath := filepath.Join(dir, "requirements.txt")
	pyprojectPath := filepath.Join(dir, "pyproject.toml")
	gemfilePath := filepath.Join(dir, "Gemfile")
	composerPath := filepath.Join(dir, "composer.json")

	if fileExists(pjPath) {
		info = detectNodeProject(dir)
	} else if fileExists(goModPath) {
		info = detectGoProject(dir)
	} else if fileExists(cargoPath) {
		info = detectRustProject(dir)
	} else if fileExists(reqsPath) || fileExists(pyprojectPath) {
		info = detectPythonProject(dir)
	} else if fileExists(gemfilePath) {
		info = detectRubyProject(dir)
	} else if fileExists(composerPath) {
		info = detectPHPProject(dir)
	}

	if info.Name == "" {
		info.Name = filepath.Base(dir)
	}

	info.KeyDirectories = detectKeyDirectories(dir)
	return info
}

func applyProjectDetection(baseContent []byte, info ProjectInfo, embeddedFS fs.FS) []byte {
	config := string(baseContent)

	config = strings.ReplaceAll(config, "{{PROJECT_NAME}}", info.Name)
	config = strings.ReplaceAll(config, "{{PROJECT_DESCRIPTION}}", info.Description)
	config = strings.ReplaceAll(config, "{{KEY_DIRECTORIES}}", formatKeyDirectories(info.KeyDirectories))
	config = strings.ReplaceAll(config, "{{TEST_COMMAND}}", info.TestCommand)
	config = strings.ReplaceAll(config, "{{TYPECHECK_COMMAND}}", info.TypecheckCommand)
	config = strings.ReplaceAll(config, "{{BUILD_COMMAND}}", info.BuildCommand)
	config = strings.ReplaceAll(config, "{{FRAMEWORK}}", info.Framework)

	// Insert language-specific template at marker
	if info.LanguageTemplate != "" {
		if langContent, err := fs.ReadFile(embeddedFS, "templates/languages/"+info.LanguageTemplate); err == nil {
			config = strings.ReplaceAll(config, "<!-- LANGUAGE_SPECIFIC -->", string(langContent))
		}
	}
	config = strings.ReplaceAll(config, "<!-- LANGUAGE_SPECIFIC -->", "")

	// Strip remaining unfilled placeholders
	config = placeholderRe.ReplaceAllString(config, "")

	// Remove empty sections and collapse blank lines
	config = removeEmptySections(config)

	return []byte(config)
}

func removeEmptySections(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "## ") {
			// Look ahead: is the section empty?
			j := i + 1
			hasContent := false
			for j < len(lines) {
				nextTrimmed := strings.TrimSpace(lines[j])
				if strings.HasPrefix(nextTrimmed, "## ") || nextTrimmed == "---" {
					break
				}
				if nextTrimmed != "" && nextTrimmed != "```" && nextTrimmed != "```bash" {
					hasContent = true
					break
				}
				j++
			}
			if !hasContent {
				i = j
				continue
			}
		}
		result = append(result, lines[i])
		i++
	}
	return collapseBlankLines(strings.Join(result, "\n"))
}

func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}
