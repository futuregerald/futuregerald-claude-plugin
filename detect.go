package main

import (
	"fmt"
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
