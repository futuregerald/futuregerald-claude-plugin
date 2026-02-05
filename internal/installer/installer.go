package installer

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
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
	Pack        string // Which pack this skill belongs to (empty = core)
	FilePath    string // Original file path
	Content     []byte
}

// Options configures the installer behavior.
type Options struct {
	Force  bool // Overwrite existing files
	DryRun bool // Don't actually write files
}

// Installer handles installing skills and agents.md.
type Installer struct {
	fs      embed.FS
	options Options
}

// New creates a new Installer with the given embedded filesystem and options.
func New(fs embed.FS, opts Options) *Installer {
	return &Installer{
		fs:      fs,
		options: opts,
	}
}

// InstallSkills installs skills to the specified directory, optionally filtered by packs.
func (i *Installer) InstallSkills(baseDir, skillsPath string, packs []string) ([]string, error) {
	var results []string
	skillsDir := filepath.Join(baseDir, skillsPath)

	// Install core skills (in skills/ root)
	if len(packs) == 0 || contains(packs, "core") {
		coreResults, err := i.installSkillsFromDir("skills", skillsDir, "")
		if err != nil {
			return nil, err
		}
		results = append(results, coreResults...)
	}

	// Install language pack skills
	packDirs, err := i.fs.ReadDir("skills")
	if err != nil {
		return nil, fmt.Errorf("reading skills directory: %w", err)
	}

	for _, entry := range packDirs {
		if !entry.IsDir() {
			continue
		}
		packName := entry.Name()
		if len(packs) > 0 && !contains(packs, packName) {
			continue
		}

		packResults, err := i.installSkillsFromDir(
			filepath.Join("skills", packName),
			filepath.Join(skillsDir, packName),
			packName,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, packResults...)
	}

	return results, nil
}

func (i *Installer) installSkillsFromDir(srcDir, destDir, pack string) ([]string, error) {
	var results []string

	entries, err := i.fs.ReadDir(srcDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := i.fs.ReadFile(filepath.Join(srcDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		targetPath := filepath.Join(destDir, entry.Name())
		result, err := i.writeFile(targetPath, content)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// InstallSkillsFiltered installs skills matching the given tags and/or languages.
func (i *Installer) InstallSkillsFiltered(baseDir, skillsPath string, tags, languages []string) ([]string, error) {
	var results []string
	skillsDir := filepath.Join(baseDir, skillsPath)

	skills, err := i.ListAllSkills()
	if err != nil {
		return nil, err
	}

	for _, skill := range skills {
		if !matchesFilter(skill, tags, languages) {
			continue
		}

		destPath := filepath.Join(skillsDir, filepath.Base(skill.FilePath))
		if skill.Pack != "" {
			destPath = filepath.Join(skillsDir, skill.Pack, filepath.Base(skill.FilePath))
		}

		result, err := i.writeFile(destPath, skill.Content)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// InstallFromLocal installs skills from a local directory.
func (i *Installer) InstallFromLocal(srcDir, destDir string) ([]string, error) {
	var results []string

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		// Preserve directory structure
		relPath, _ := filepath.Rel(srcDir, path)
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
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "skill-installer-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone the repo
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("cloning repo: %s: %w", string(output), err)
	}

	// Look for skills directory or use root
	skillsDir := tmpDir
	if info, err := os.Stat(filepath.Join(tmpDir, "skills")); err == nil && info.IsDir() {
		skillsDir = filepath.Join(tmpDir, "skills")
	}

	return i.InstallFromLocal(skillsDir, destDir)
}

// InstallFromURL downloads and extracts a tarball of skills.
func (i *Installer) InstallFromURL(url, destDir string) ([]string, error) {
	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: status %d", url, resp.StatusCode)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "skill-installer-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract tarball
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

// InstallAgentsMD installs agents.md to the specified directory.
func (i *Installer) InstallAgentsMD(targetDir string) (string, error) {
	content, err := i.fs.ReadFile("templates/agents.md")
	if err != nil {
		return "", fmt.Errorf("reading agents.md template: %w", err)
	}

	targetPath := filepath.Join(targetDir, "agents.md")
	return i.writeFile(targetPath, content)
}

// InstallAgents installs agent files to the specified directory.
func (i *Installer) InstallAgents(targetDir string) ([]string, error) {
	var results []string

	entries, err := i.fs.ReadDir("agents")
	if err != nil {
		// No agents directory is not an error
		return results, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := i.fs.ReadFile(filepath.Join("agents", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading agent %s: %w", entry.Name(), err)
		}

		targetPath := filepath.Join(targetDir, entry.Name())
		result, err := i.writeFile(targetPath, content)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// ListAgents returns metadata about available agents.
func (i *Installer) ListAgents() ([]Skill, error) {
	entries, err := i.fs.ReadDir("agents")
	if err != nil {
		// No agents directory is not an error
		return nil, nil
	}

	var agents []Skill
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join("agents", entry.Name())
		content, err := i.fs.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		agent, err := parseSkill(content)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		agent.FilePath = filePath
		agents = append(agents, agent)
	}

	return agents, nil
}

func (i *Installer) writeFile(path string, content []byte) (string, error) {
	exists := fileExists(path)

	if exists && !i.options.Force {
		return fmt.Sprintf("SKIP: %s (already exists, use --force to overwrite)", path), nil
	}

	if i.options.DryRun {
		if exists {
			return fmt.Sprintf("WOULD OVERWRITE: %s", path), nil
		}
		return fmt.Sprintf("WOULD CREATE: %s", path), nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	if exists {
		return fmt.Sprintf("UPDATED: %s", path), nil
	}
	return fmt.Sprintf("CREATED: %s", path), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ListSkills returns metadata about core skills only (for backward compatibility).
func (i *Installer) ListSkills() ([]Skill, error) {
	return i.listSkillsFromDir("skills", "")
}

// ListAllSkills returns metadata about all skills including language packs.
func (i *Installer) ListAllSkills() ([]Skill, error) {
	var allSkills []Skill

	// Core skills
	coreSkills, err := i.listSkillsFromDir("skills", "")
	if err != nil {
		return nil, err
	}
	allSkills = append(allSkills, coreSkills...)

	// Language pack skills
	entries, err := i.fs.ReadDir("skills")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packSkills, err := i.listSkillsFromDir(filepath.Join("skills", entry.Name()), entry.Name())
		if err != nil {
			return nil, err
		}
		allSkills = append(allSkills, packSkills...)
	}

	return allSkills, nil
}

// ListPacks returns available language packs.
func (i *Installer) ListPacks() ([]string, error) {
	var packs []string

	entries, err := i.fs.ReadDir("skills")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			packs = append(packs, entry.Name())
		}
	}

	return packs, nil
}

func (i *Installer) listSkillsFromDir(dir, pack string) ([]Skill, error) {
	entries, err := i.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	var skills []Skill
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		content, err := i.fs.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		skill, err := parseSkill(content)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		skill.Pack = pack
		skill.FilePath = filePath
		skills = append(skills, skill)
	}

	return skills, nil
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
			skill.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
		} else if strings.HasPrefix(trimmed, "model:") {
			skill.Model = strings.TrimSpace(strings.TrimPrefix(trimmed, "model:"))
		} else if strings.HasPrefix(trimmed, "tags:") {
			skill.Tags = parseYAMLList(strings.TrimPrefix(trimmed, "tags:"))
		} else if strings.HasPrefix(trimmed, "languages:") {
			skill.Languages = parseYAMLList(strings.TrimPrefix(trimmed, "languages:"))
		}
	}

	if skill.Name == "" {
		return skill, fmt.Errorf("skill missing name in frontmatter")
	}

	return skill, nil
}

// parseYAMLList parses a simple YAML list like [a, b, c] or - a format.
func parseYAMLList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Handle [a, b, c] format
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
		// "any" matches all
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
		strings.Title(strings.ReplaceAll(name, "-", " ")),
		description)
}
