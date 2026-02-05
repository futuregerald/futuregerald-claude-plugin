package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// Skill represents a skill definition parsed from frontmatter.
type Skill struct {
	Name        string
	Description string
	Model       string
	Tags        []string
	Languages   []string
	DirPath     string // Directory path within embedded FS (e.g., "skills/systematic-debugging")
	FilePath    string // SKILL.md path within embedded FS
	Content     []byte // Content of SKILL.md
}

// Options configures the installer behavior.
type Options struct {
	Force  bool // Overwrite existing files
	DryRun bool // Don't actually write files
}

// AgentNameFunc transforms an agent filename for the target framework.
// Pass nil to keep original names.
type AgentNameFunc func(originalName string) string

// Installer handles installing skills, agents, and commands.
type Installer struct {
	fsys    fs.FS
	options Options
}

// New creates a new Installer with the given filesystem and options.
func New(fsys fs.FS, opts Options) *Installer {
	return &Installer{
		fsys:    fsys,
		options: opts,
	}
}

// discoverSkills walks the skills/ directory finding directories that contain SKILL.md.
func (i *Installer) discoverSkills() ([]Skill, error) {
	var skills []Skill

	entries, err := fs.ReadDir(i.fsys, "skills")
	if err != nil {
		return nil, fmt.Errorf("reading skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := path.Join("skills", entry.Name())
		skill, ok := i.tryParseSkillDir(dirPath)
		if ok {
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// tryParseSkillDir attempts to parse a skill directory by reading SKILL.md.
func (i *Installer) tryParseSkillDir(dirPath string) (Skill, bool) {
	// Try SKILL.md (uppercase first)
	skillMD := path.Join(dirPath, "SKILL.md")
	content, err := fs.ReadFile(i.fsys, skillMD)
	if err != nil {
		// Try skill.md (lowercase fallback)
		skillMD = path.Join(dirPath, "skill.md")
		content, err = fs.ReadFile(i.fsys, skillMD)
		if err != nil {
			return Skill{}, false
		}
	}

	skill, err := parseSkill(content)
	if err != nil {
		// If frontmatter parse fails, use directory name as name
		skill = Skill{
			Name:    path.Base(dirPath),
			Content: content,
		}
	}
	skill.DirPath = dirPath
	skill.FilePath = skillMD

	if skill.Model == "" {
		skill.Model = "sonnet"
	}

	return skill, true
}

// listDirFiles returns all file paths under a directory in the embedded FS.
func (i *Installer) listDirFiles(dirPath string) ([]string, error) {
	var files []string
	err := fs.WalkDir(i.fsys, dirPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		files = append(files, p)
		return nil
	})
	return files, err
}

// InstallSkills copies entire skill directories to destDir, optionally filtered by tags/languages.
func (i *Installer) InstallSkills(destDir string, tags, languages []string) ([]string, error) {
	var results []string

	skills, err := i.discoverSkills()
	if err != nil {
		return nil, err
	}

	for _, skill := range skills {
		// Apply tag/language filter if provided
		if len(tags) > 0 || len(languages) > 0 {
			if !matchesFilter(skill, tags, languages) {
				continue
			}
		}

		// List all files in this skill's directory
		files, err := i.listDirFiles(skill.DirPath)
		if err != nil {
			return nil, fmt.Errorf("listing files in %s: %w", skill.DirPath, err)
		}

		for _, file := range files {
			// Compute relative path from "skills/" prefix
			relPath := strings.TrimPrefix(file, "skills/")
			targetPath := filepath.Join(destDir, relPath)

			fileContent, err := fs.ReadFile(i.fsys, file)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", file, err)
			}

			result, err := i.writeFile(targetPath, fileContent)
			if err != nil {
				return nil, err
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// InstallAgents copies agent .md files to destDir with optional renaming.
func (i *Installer) InstallAgents(destDir string, nameFunc AgentNameFunc) ([]string, error) {
	var results []string

	entries, err := fs.ReadDir(i.fsys, "agents")
	if err != nil {
		return results, nil // No agents directory is not an error
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := fs.ReadFile(i.fsys, path.Join("agents", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading agent %s: %w", entry.Name(), err)
		}

		destName := entry.Name()
		if nameFunc != nil {
			destName = nameFunc(destName)
		}

		targetPath := filepath.Join(destDir, destName)
		result, err := i.writeFile(targetPath, content)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// CopilotAgentName converts "debugger.md" to "debugger.agent.md".
func CopilotAgentName(name string) string {
	return strings.TrimSuffix(name, ".md") + ".agent.md"
}

// InstallCommands copies command directories to destDir.
func (i *Installer) InstallCommands(destDir string) ([]string, error) {
	var results []string

	files, err := i.listDirFiles("commands")
	if err != nil {
		return results, nil // No commands directory is not an error
	}

	for _, file := range files {
		relPath := strings.TrimPrefix(file, "commands/")
		targetPath := filepath.Join(destDir, relPath)

		content, err := fs.ReadFile(i.fsys, file)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}

		result, err := i.writeFile(targetPath, content)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// ListSkills returns metadata about all discovered skills.
func (i *Installer) ListSkills() ([]Skill, error) {
	return i.discoverSkills()
}

// ListAllSkills is an alias for ListSkills (kept for backward compat).
func (i *Installer) ListAllSkills() ([]Skill, error) {
	return i.discoverSkills()
}

// InstallFromLocal installs skills from a local directory (copies all files preserving structure).
func (i *Installer) InstallFromLocal(srcDir, destDir string) ([]string, error) {
	var results []string

	err := filepath.WalkDir(srcDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		content, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}

		relPath, _ := filepath.Rel(srcDir, p)
		targetPath := filepath.Join(destDir, relPath)

		result, err := i.writeFile(targetPath, content)
		if err != nil {
			return err
		}
		results = append(results, result)
		return nil
	})

	return results, err
}

// InstallFromGit clones a git repo and installs skills from it.
func (i *Installer) InstallFromGit(repoURL, destDir string) ([]string, error) {
	tmpDir, err := os.MkdirTemp("", "skill-installer-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("cloning repo: %s: %w", string(output), err)
	}

	skillsDir := tmpDir
	if info, err := os.Stat(filepath.Join(tmpDir, "skills")); err == nil && info.IsDir() {
		skillsDir = filepath.Join(tmpDir, "skills")
	}

	return i.InstallFromLocal(skillsDir, destDir)
}

// InstallFromURL downloads and extracts a tarball of skills.
func (i *Installer) InstallFromURL(url, destDir string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: status %d", url, resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "skill-installer-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(resp.Body, tmpDir); err != nil {
		return nil, fmt.Errorf("extracting archive: %w", err)
	}

	return i.InstallFromLocal(tmpDir, destDir)
}

func extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		// Prevent zip-slip: ensure target stays within destDir
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func (i *Installer) writeFile(filePath string, content []byte) (string, error) {
	exists := fileExists(filePath)

	if exists && !i.options.Force {
		return fmt.Sprintf("SKIP: %s (already exists, use --force to overwrite)", filePath), nil
	}

	if i.options.DryRun {
		if exists {
			return fmt.Sprintf("WOULD OVERWRITE: %s", filePath), nil
		}
		return fmt.Sprintf("WOULD CREATE: %s", filePath), nil
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", fmt.Errorf("writing %s: %w", filePath, err)
	}

	if exists {
		return fmt.Sprintf("UPDATED: %s", filePath), nil
	}
	return fmt.Sprintf("CREATED: %s", filePath), nil
}

func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	return err == nil && !info.IsDir()
}

// parseSkill extracts skill metadata from frontmatter.
func parseSkill(content []byte) (Skill, error) {
	lines := strings.Split(string(content), "\n")

	var skill Skill
	skill.Content = content

	inFrontmatter := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}

		if !inFrontmatter {
			continue
		}

		if strings.HasPrefix(trimmed, "name:") {
			skill.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		} else if strings.HasPrefix(trimmed, "description:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			skill.Description = val
		} else if strings.HasPrefix(trimmed, "model:") {
			skill.Model = strings.TrimSpace(strings.TrimPrefix(trimmed, "model:"))
		} else if strings.HasPrefix(trimmed, "tags:") {
			skill.Tags = parseYAMLList(strings.TrimPrefix(trimmed, "tags:"))
		} else if strings.HasPrefix(trimmed, "languages:") {
			skill.Languages = parseYAMLList(strings.TrimPrefix(trimmed, "languages:"))
		}
		// Ignore unknown fields (allowed-tools, argument-hint, etc.)
	}

	if skill.Name == "" {
		return skill, fmt.Errorf("skill missing name in frontmatter")
	}
	return skill, nil
}

// parseYAMLList parses a simple YAML list like [a, b, c].
func parseYAMLList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = strings.TrimPrefix(s, "[")
		s = strings.TrimSuffix(s, "]")
		parts := strings.Split(s, ",")
		var result []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}

	return []string{s}
}

func matchesFilter(skill Skill, tags, languages []string) bool {
	if len(tags) == 0 && len(languages) == 0 {
		return true
	}

	if len(tags) > 0 {
		found := false
		for _, t := range tags {
			if contains(skill.Tags, t) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(languages) > 0 {
		if contains(skill.Languages, "any") {
			return true
		}
		found := false
		for _, l := range languages {
			if contains(skill.Languages, l) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// titleCase capitalizes the first letter of a string.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// GenerateSkillTemplate creates a new skill file with proper frontmatter.
func GenerateSkillTemplate(name, description, model string, tags, languages []string) string {
	tagsStr := "[" + strings.Join(tags, ", ") + "]"
	langsStr := "[" + strings.Join(languages, ", ") + "]"

	return fmt.Sprintf(`---
name: %s
description: %s
model: %s
tags: %s
languages: %s
---

# %s

You are a specialized agent for %s.

## Capabilities

- [List what this skill can do]

## Guidelines

1. [First guideline]
2. [Second guideline]

## Output Format

[Describe expected output format]

## Tools to Use

- **Read** - Read files
- **Grep** - Search code
- **Glob** - Find files

## Do NOT

- [Things to avoid]
`, name, description, model, tagsStr, langsStr,
		titleCase(strings.ReplaceAll(name, "-", " ")),
		description)
}
