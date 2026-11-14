package installer

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	HarnessClaude      = "claude"
	HarnessAntigravity = "antigravity"
	HarnessGemini      = "gemini"
)

func IsGeminiHarness(harness string) bool {
	return harness == HarnessAntigravity || harness == HarnessGemini
}

func isAntigravityOnlySkill(skillName string) bool {
	return skillName == "antigravity-orchestration" || skillName == "peer-pipeline-swarm" || skillName == "baton-pass"
}

func UnlinkIfSymlink(destPath string, dryRun bool) (bool, error) {
	info, err := os.Lstat(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}
	if dryRun {
		return true, nil
	}
	if err := os.Remove(destPath); err != nil {
		return false, fmt.Errorf("unlinking symlink %s: %w", destPath, err)
	}
	return true, nil
}

func shouldSkipSourcePath(relSlash string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(relSlash), "./")
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" {
		return false
	}
	lower := strings.ToLower(clean)
	if strings.HasSuffix(lower, ".pyc") || strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".local.md") {
		return true
	}
	if clean == "references/config.md" || strings.HasSuffix(clean, "/references/config.md") {
		return true
	}
	if strings.HasPrefix(clean, "shared/skills/codebase-memory-") || strings.HasPrefix(clean, "skills/codebase-memory-") || strings.HasPrefix(clean, "codebase-memory-") {
		return true
	}
	base := path.Base(clean)
	if (strings.HasPrefix(base, "test_") && strings.HasSuffix(lower, ".py")) || strings.HasSuffix(lower, "_test.py") {
		return true
	}
	segments := strings.Split(clean, "/")
	for _, seg := range segments {
		switch seg {
		case "__pycache__", ".pytest_cache", ".ruff_cache", ".mypy_cache", ".venv", ".trash", ".DS_Store":
			return true
		}
	}
	return false
}

func isPreservedLocalFile(filePath string) bool {
	slashed := filepath.ToSlash(filePath)
	return strings.HasSuffix(slashed, ".local.md") || strings.HasSuffix(slashed, "/references/config.md")
}

func AdaptMarkdownForGemini(content []byte, harness string, isGlobal bool) []byte {
	text := string(content)
	text = stripFrontmatterModelAndEffort(text)

	globalSkills := "~/.gemini/config/skills"
	globalSkillsHome := "$HOME/.gemini/config/skills"
	scopedSkills := ".agents/skills"
	globalAgents := "~/.gemini/config/plugins/antigravity-subagents/agents"
	globalAgentsHome := "$HOME/.gemini/config/plugins/antigravity-subagents/agents"
	scopedAgents := ".agents/agents"

	if harness == HarnessGemini {
		globalSkills = "~/.gemini/skills"
		globalSkillsHome = "$HOME/.gemini/skills"
		scopedSkills = ".gemini/skills"
		globalAgents = "~/.gemini/agents"
		globalAgentsHome = "$HOME/.gemini/agents"
		scopedAgents = ".gemini/agents"
	} else if isGlobal {
		scopedSkills = globalSkills
		scopedAgents = globalAgents
	}

	if harness == HarnessGemini && isGlobal {
		scopedSkills = globalSkills
		scopedAgents = globalAgents
	}

	replacer := strings.NewReplacer(
		"$HOME/.claude/point-of-view", "$HOME/.gemini/point-of-view",
		"~/.claude/point-of-view", "~/.gemini/point-of-view",
		"$HOME/.claude/skills", globalSkillsHome,
		"~/.claude/skills", globalSkills,
		"$HOME/.claude/agents", globalAgentsHome,
		"~/.claude/agents", globalAgents,
		"$HOME/.claude/CLAUDE.md", "$HOME/.gemini/GEMINI.md",
		"~/.claude/CLAUDE.md", "~/.gemini/GEMINI.md",
		".claude/CLAUDE.md", "GEMINI.md",
		".claude/skills", scopedSkills,
		".claude/agents", scopedAgents,
		"TaskCreate", "task.md",
	)
	text = replacer.Replace(text)

	if harness == HarnessGemini {
		text = adaptOverlayPathsForGeminiCLI(text)
	}

	lines := strings.Split(text, "\n")
	inFrontmatter := false
	frontmatterDone := false
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !frontmatterDone && idx == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
				frontmatterDone = true
			}
			continue
		}
		if strings.Contains(line, "GEMINI.md") {
			continue
		}
		lines[idx] = replaceStandaloneClaudeMD(line)
	}

	return []byte(strings.Join(lines, "\n"))
}

func adaptOverlayPathsForGeminiCLI(text string) string {
	r := strings.NewReplacer(
		"$HOME/.gemini/config/skills", "$HOME/.gemini/skills",
		"~/.gemini/config/skills", "~/.gemini/skills",
		"$HOME/.gemini/config/plugins/antigravity-subagents/agents", "$HOME/.gemini/agents",
		"~/.gemini/config/plugins/antigravity-subagents/agents", "~/.gemini/agents",
	)
	return r.Replace(text)
}

func stripFrontmatterModelAndEffort(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return text
	}
	var out []string
	out = append(out, lines[0])
	inFrontmatter := true
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
				out = append(out, lines[i])
				continue
			}
			if strings.HasPrefix(trimmed, "model:") || strings.HasPrefix(trimmed, "effort:") {
				continue
			}
		}
		out = append(out, lines[i])
	}
	return strings.Join(out, "\n")
}

func isTokenBoundaryChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

func replaceStandaloneClaudeMD(line string) string {
	const needle = "CLAUDE.md"
	if !strings.Contains(line, needle) {
		return line
	}
	var b strings.Builder
	i := 0
	for i < len(line) {
		pos := strings.Index(line[i:], needle)
		if pos < 0 {
			b.WriteString(line[i:])
			break
		}
		absPos := i + pos
		endPos := absPos + len(needle)
		validStart := absPos == 0 || !isTokenBoundaryChar(line[absPos-1])
		validEnd := endPos == len(line) || !isTokenBoundaryChar(line[endPos])
		if validStart && validEnd {
			b.WriteString(line[i:absPos])
			b.WriteString("GEMINI.md")
			i = endPos
		} else {
			b.WriteString(line[i : absPos+len(needle)])
			i = endPos
		}
	}
	return b.String()
}

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

// HasForce returns whether the installer has Force enabled.
func (i *Installer) HasForce() bool {
	return i.options.Force
}

func (i *Installer) discoverSkills() ([]Skill, error) {
	if _, err := fs.Stat(i.fsys, "shared/skills"); err == nil {
		return i.discoverSkillsInDir("shared/skills")
	}
	return i.discoverSkillsInDir("skills")
}

func trimSkillsPrefix(p string) string {
	if strings.HasPrefix(p, "shared/skills/") {
		return strings.TrimPrefix(p, "shared/skills/")
	}
	return strings.TrimPrefix(p, "skills/")
}

func (i *Installer) discoverSkillsInDir(rootDir string) ([]Skill, error) {
	var skills []Skill

	entries, err := fs.ReadDir(i.fsys, rootDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s directory: %w", rootDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := path.Join(rootDir, entry.Name())
		if shouldSkipSourcePath(dirPath) {
			continue
		}
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
		if err != nil {
			return err
		}
		if shouldSkipSourcePath(p) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	return files, err
}

// InstallSkills copies entire skill directories to destDir, optionally filtered by tags/languages.
func (i *Installer) InstallSkills(destDir string, tags, languages []string) ([]string, error) {
	return i.InstallSkillsForHarness(destDir, HarnessClaude, false, tags, languages)
}

func (i *Installer) InstallSkillsForHarness(destDir, harness string, isGlobal bool, tags, languages []string) ([]string, error) {
	var results []string

	unlinked, err := UnlinkIfSymlink(destDir, i.options.DryRun)
	if err != nil {
		return nil, err
	}
	if unlinked && i.options.DryRun {
		results = append(results, fmt.Sprintf("WOULD UNLINK SYMLINK: %s", destDir))
	}

	skills, err := i.discoverSkills()
	if err != nil {
		return nil, err
	}

	if !IsGeminiHarness(harness) {
		for _, skill := range skills {
			if len(tags) > 0 || len(languages) > 0 {
				if !matchesFilter(skill, tags, languages) {
					continue
				}
			}

			files, err := i.listDirFiles(skill.DirPath)
			if err != nil {
				return nil, fmt.Errorf("listing files in %s: %w", skill.DirPath, err)
			}

			for _, file := range files {
				relPath := trimSkillsPrefix(file)
				targetPath := filepath.Join(destDir, relPath)

				fileContent, err := fs.ReadFile(i.fsys, file)
				if err != nil {
					return nil, fmt.Errorf("reading %s: %w", file, err)
				}

				result, err := i.writeFileWithRoot(destDir, targetPath, fileContent, 0644)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
			}
		}
		return results, nil
	}

	baseByDirName := make(map[string]Skill, len(skills))
	var orderedNames []string
	seenNames := make(map[string]bool)
	for _, s := range skills {
		dirName := path.Base(s.DirPath)
		baseByDirName[dirName] = s
		if !seenNames[dirName] {
			seenNames[dirName] = true
			orderedNames = append(orderedNames, dirName)
		}
	}

	overrideByDirName := make(map[string]Skill)
	overrideFilesByRel := make(map[string]bool)
	if _, statErr := fs.Stat(i.fsys, "antigravity/skills"); statErr == nil {
		overSkills, overErr := i.discoverSkillsInDir("antigravity/skills")
		if overErr == nil {
			for _, override := range overSkills {
				dirName := path.Base(override.DirPath)
				if harness == HarnessGemini && isAntigravityOnlySkill(dirName) {
					continue
				}
				if base, hasBase := baseByDirName[dirName]; hasBase {
					if len(override.Tags) == 0 {
						override.Tags = base.Tags
					}
					if len(override.Languages) == 0 {
						override.Languages = base.Languages
					}
				}
				overrideByDirName[dirName] = override
				if !seenNames[dirName] {
					seenNames[dirName] = true
					orderedNames = append(orderedNames, dirName)
				}
				oFiles, fErr := i.listDirFiles(override.DirPath)
				if fErr == nil {
					for _, of := range oFiles {
						rel := strings.TrimPrefix(of, "antigravity/skills/")
						overrideFilesByRel[rel] = true
					}
				}
			}
		}
	}

	for _, dirName := range orderedNames {
		if harness == HarnessGemini && isAntigravityOnlySkill(dirName) {
			continue
		}
		baseSkill, hasBase := baseByDirName[dirName]
		overSkill, hasOver := overrideByDirName[dirName]

		filterSkill := baseSkill
		if hasOver {
			filterSkill = overSkill
		}
		if len(tags) > 0 || len(languages) > 0 {
			if !matchesFilter(filterSkill, tags, languages) {
				continue
			}
		}

		if hasBase {
			files, err := i.listDirFiles(baseSkill.DirPath)
			if err != nil {
				return nil, fmt.Errorf("listing files in %s: %w", baseSkill.DirPath, err)
			}
			for _, file := range files {
				relPath := trimSkillsPrefix(file)
				if overrideFilesByRel[relPath] {
					continue
				}
				fileContent, err := fs.ReadFile(i.fsys, file)
				if err != nil {
					return nil, fmt.Errorf("reading %s: %w", file, err)
				}
				if strings.HasSuffix(strings.ToLower(relPath), ".md") {
					fileContent = AdaptMarkdownForGemini(fileContent, harness, isGlobal)
				}
				targetPath := filepath.Join(destDir, relPath)
				result, err := i.writeFileWithRoot(destDir, targetPath, fileContent, 0644)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
			}
		}

		if hasOver {
			files, err := i.listDirFiles(overSkill.DirPath)
			if err != nil {
				return nil, fmt.Errorf("listing files in %s: %w", overSkill.DirPath, err)
			}
			for _, file := range files {
				relPath := strings.TrimPrefix(file, "antigravity/skills/")
				fileContent, err := fs.ReadFile(i.fsys, file)
				if err != nil {
					return nil, fmt.Errorf("reading %s: %w", file, err)
				}
				if harness == HarnessGemini && strings.HasSuffix(strings.ToLower(relPath), ".md") {
					fileContent = []byte(adaptOverlayPathsForGeminiCLI(string(fileContent)))
				}
				targetPath := filepath.Join(destDir, relPath)
				result, err := i.writeFileWithRoot(destDir, targetPath, fileContent, 0644)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// InstallAgents copies agent .md files to destDir with optional renaming.
func (i *Installer) InstallAgents(destDir string, nameFunc AgentNameFunc) ([]string, error) {
	return i.InstallAgentsForHarness(destDir, HarnessClaude, false, nameFunc)
}

func (i *Installer) InstallAgentsForHarness(destDir, harness string, isGlobal bool, nameFunc AgentNameFunc) ([]string, error) {
	var results []string

	unlinked, err := UnlinkIfSymlink(destDir, i.options.DryRun)
	if err != nil {
		return nil, err
	}
	if unlinked && i.options.DryRun {
		results = append(results, fmt.Sprintf("WOULD UNLINK SYMLINK: %s", destDir))
	}

	agentsDir := "claude/agents"
	if _, err := fs.Stat(i.fsys, agentsDir); err != nil {
		agentsDir = "agents"
	}
	if IsGeminiHarness(harness) {
		agentsDir = "antigravity/plugins/antigravity-subagents/agents"
	} else if harness == "opencode" {
		if _, err := fs.Stat(i.fsys, "opencode/agents"); err == nil {
			agentsDir = "opencode/agents"
		}
	}

	entries, err := fs.ReadDir(i.fsys, agentsDir)
	if err != nil {
		return results, nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		srcPath := path.Join(agentsDir, entry.Name())
		if shouldSkipSourcePath(srcPath) {
			continue
		}

		content, err := fs.ReadFile(i.fsys, srcPath)
		if err != nil {
			return nil, fmt.Errorf("reading agent %s: %w", entry.Name(), err)
		}
		if harness == HarnessGemini {
			content = []byte(adaptOverlayPathsForGeminiCLI(string(content)))
		}

		destName := entry.Name()
		if nameFunc != nil {
			destName = nameFunc(destName)
		}

		targetPath := filepath.Join(destDir, destName)
		result, err := i.writeFileWithRoot(destDir, targetPath, content, 0644)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if harness == HarnessAntigravity && isGlobal {
		pluginSrc := "antigravity/plugins/antigravity-subagents/plugin.json"
		if pluginBytes, err := fs.ReadFile(i.fsys, pluginSrc); err == nil {
			pluginDir := filepath.Dir(destDir)
			pluginDest := filepath.Join(pluginDir, "plugin.json")
			res, err := i.writeFileWithRoot(pluginDir, pluginDest, pluginBytes, 0644)
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}
	}

	return results, nil
}

func (i *Installer) InstallGlobalHarnessAssets(geminiHome, harness string) ([]string, error) {
	if !IsGeminiHarness(harness) {
		return nil, nil
	}

	var results []string
	if info, err := os.Lstat(geminiHome); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to modify symlinked gemini home directory %s", geminiHome)
	}
	configDir := filepath.Join(geminiHome, "config")
	if info, err := os.Lstat(configDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refusing to modify symlinked config directory %s", configDir)
	}

	if geminiMD, err := fs.ReadFile(i.fsys, "antigravity/GEMINI.md"); err == nil {
		targetMD := filepath.Join(geminiHome, "GEMINI.md")
		if harness == HarnessGemini {
			if _, lstatErr := os.Lstat(targetMD); lstatErr == nil {
				results = append(results, fmt.Sprintf("SKIP: %s (shared global GEMINI.md already exists)", targetMD))
			} else {
				res, err := i.writeFileWithRoot(targetMD, targetMD, geminiMD, 0644)
				if err != nil {
					return nil, err
				}
				results = append(results, res)
			}
		} else {
			res, err := i.writeFileWithRoot(targetMD, targetMD, geminiMD, 0644)
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}
	}

	if orgEx, err := fs.ReadFile(i.fsys, "antigravity/rules/org.example.md"); err == nil {
		rulesDir := filepath.Join(configDir, "rules")
		if info, err := os.Lstat(rulesDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing to modify symlinked rules directory %s", rulesDir)
		}
		targetOrg := filepath.Join(rulesDir, "org.example.md")
		res, err := i.writeFileWithRoot(targetOrg, targetOrg, orgEx, 0644)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	if harness == HarnessAntigravity {
		hooksDest := filepath.Join(configDir, "hooks")
		unlinkedHooks, err := UnlinkIfSymlink(hooksDest, i.options.DryRun)
		if err != nil {
			return nil, err
		}
		if unlinkedHooks && i.options.DryRun {
			results = append(results, fmt.Sprintf("WOULD UNLINK SYMLINK: %s", hooksDest))
		}

		if _, err := fs.Stat(i.fsys, "antigravity/hooks"); err == nil {
			hookFiles, err := i.listDirFiles("antigravity/hooks")
			if err != nil {
				return nil, fmt.Errorf("listing antigravity/hooks: %w", err)
			}
			for _, hf := range hookFiles {
				base := path.Base(hf)
				if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") {
					continue
				}
				rel := strings.TrimPrefix(hf, "antigravity/hooks/")
				data, err := fs.ReadFile(i.fsys, hf)
				if err != nil {
					return nil, fmt.Errorf("reading %s: %w", hf, err)
				}
				mode := os.FileMode(0644)
				if strings.HasSuffix(base, ".sh") {
					mode = 0755
				}
				targetPath := filepath.Join(hooksDest, rel)
				res, err := i.writeFileWithRoot(hooksDest, targetPath, data, mode)
				if err != nil {
					return nil, err
				}
				results = append(results, res)
			}
		}

		if hooksJSONBytes, err := fs.ReadFile(i.fsys, "antigravity/hooks.json"); err == nil {
			hooksJSONDest := filepath.Join(configDir, "hooks.json")
			res, err := i.mergeHooksJSON(hooksJSONDest, hooksJSONBytes)
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}
	}

	return results, nil
}

func (i *Installer) mergeHooksJSON(hooksJSONPath string, srcBytes []byte) (string, error) {
	var srcMap map[string]any
	if err := json.Unmarshal(srcBytes, &srcMap); err != nil {
		return "", fmt.Errorf("parsing embedded hooks.json: %w", err)
	}

	parentDir := filepath.Dir(hooksJSONPath)
	if info, err := os.Lstat(parentDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("refusing to modify symlinked config directory %s", parentDir)
	}

	var existingBytes []byte
	exists := false
	if statInfo, statErr := os.Stat(hooksJSONPath); statErr == nil && !statInfo.IsDir() {
		b, rErr := os.ReadFile(hooksJSONPath)
		if rErr != nil {
			return "", fmt.Errorf("reading existing %s: %w", hooksJSONPath, rErr)
		}
		existingBytes = b
		exists = true
	}

	if !exists {
		if _, err := UnlinkIfSymlink(hooksJSONPath, i.options.DryRun); err != nil {
			return "", err
		}
		formatted, err := json.MarshalIndent(srcMap, "", "  ")
		if err != nil {
			return "", err
		}
		formatted = append(formatted, '\n')
		return i.writeFileWithRoot(hooksJSONPath, hooksJSONPath, formatted, 0644)
	}

	var existingMap map[string]any
	if err := json.Unmarshal(existingBytes, &existingMap); err != nil {
		return "", fmt.Errorf("parsing existing %s: %w", hooksJSONPath, err)
	}
	if existingMap == nil {
		existingMap = make(map[string]any)
	}

	needsWrite := false
	for k, v := range srcMap {
		if _, hasKey := existingMap[k]; !hasKey || i.options.Force {
			existingMap[k] = v
			needsWrite = true
		}
	}

	if !needsWrite {
		return fmt.Sprintf("SKIP: %s (already exists, use --force to overwrite)", hooksJSONPath), nil
	}

	if i.options.DryRun {
		return fmt.Sprintf("WOULD OVERWRITE: %s", hooksJSONPath), nil
	}

	if _, err := UnlinkIfSymlink(hooksJSONPath, false); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(existingMap, "", "  ")
	if err != nil {
		return "", err
	}
	formatted = append(formatted, '\n')
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", parentDir, err)
	}
	if err := os.WriteFile(hooksJSONPath, formatted, 0644); err != nil {
		return "", fmt.Errorf("writing %s: %w", hooksJSONPath, err)
	}
	return fmt.Sprintf("UPDATED: %s", hooksJSONPath), nil
}

// CopilotAgentName converts "debugger.md" to "debugger.agent.md".
func CopilotAgentName(name string) string {
	return strings.TrimSuffix(name, ".md") + ".agent.md"
}

// InstallCommands copies command directories to destDir.
func (i *Installer) InstallCommands(destDir string) ([]string, error) {
	var results []string

	unlinked, err := UnlinkIfSymlink(destDir, i.options.DryRun)
	if err != nil {
		return nil, err
	}
	if unlinked && i.options.DryRun {
		results = append(results, fmt.Sprintf("WOULD UNLINK SYMLINK: %s", destDir))
	}

	commandsRoot := "claude/commands"
	if _, err := fs.Stat(i.fsys, commandsRoot); err != nil {
		commandsRoot = "commands"
	}
	files, err := i.listDirFiles(commandsRoot)
	if err != nil {
		return results, nil
	}

	for _, file := range files {
		relPath := strings.TrimPrefix(file, commandsRoot+"/")
		targetPath := filepath.Join(destDir, relPath)

		content, err := fs.ReadFile(i.fsys, file)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}

		result, err := i.writeFileWithRoot(destDir, targetPath, content, 0644)
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

func (i *Installer) ListSkillsForHarness(harness string) ([]Skill, error) {
	baseSkills, err := i.discoverSkills()
	if err != nil {
		return nil, err
	}
	if !IsGeminiHarness(harness) {
		return baseSkills, nil
	}

	byDir := make(map[string]Skill, len(baseSkills))
	var order []string
	for _, s := range baseSkills {
		d := path.Base(s.DirPath)
		byDir[d] = s
		order = append(order, d)
	}

	if _, statErr := fs.Stat(i.fsys, "antigravity/skills"); statErr == nil {
		overSkills, overErr := i.discoverSkillsInDir("antigravity/skills")
		if overErr == nil {
			for _, override := range overSkills {
				d := path.Base(override.DirPath)
				if harness == HarnessGemini && isAntigravityOnlySkill(d) {
					continue
				}
				if base, hasBase := byDir[d]; hasBase {
					if len(override.Tags) == 0 {
						override.Tags = base.Tags
					}
					if len(override.Languages) == 0 {
						override.Languages = base.Languages
					}
				} else {
					order = append(order, d)
				}
				byDir[d] = override
			}
		}
	}

	sort.Strings(order)
	var merged []Skill
	for _, d := range order {
		if harness == HarnessGemini && isAntigravityOnlySkill(d) {
			continue
		}
		merged = append(merged, byDir[d])
	}
	return merged, nil
}

// InstallFromLocal installs skills from a local directory (copies all files preserving structure).
func (i *Installer) InstallFromLocal(srcDir, destDir string) ([]string, error) {
	var results []string

	unlinked, err := UnlinkIfSymlink(destDir, i.options.DryRun)
	if err != nil {
		return nil, err
	}
	if unlinked && i.options.DryRun {
		results = append(results, fmt.Sprintf("WOULD UNLINK SYMLINK: %s", destDir))
	}

	err = filepath.WalkDir(srcDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		relPath, relErr := filepath.Rel(srcDir, p)
		if relErr == nil && relPath != "." && shouldSkipSourcePath(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}

		targetPath := filepath.Join(destDir, relPath)

		result, err := i.writeFileWithRoot(destDir, targetPath, content, 0644)
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

	cmd := exec.Command("git", "clone", "--depth", "1", "--", repoURL, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("cloning repo: %s: %w", string(output), err)
	}

	skillsDir := tmpDir
	if info, err := os.Lstat(filepath.Join(tmpDir, "skills")); err == nil && info.Mode()&os.ModeSymlink == 0 && info.IsDir() {
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
	return i.writeFileWithRoot(filepath.Dir(filePath), filePath, content, 0644)
}

func (i *Installer) writeFileWithRoot(rootDir, filePath string, content []byte, mode os.FileMode) (string, error) {
	if isPreservedLocalFile(filePath) {
		if _, err := os.Lstat(filePath); err == nil {
			return fmt.Sprintf("SKIP: %s (preserving local config)", filePath), nil
		}
	}

	symlinkInChain, err := i.checkOrCleanSymlinkChain(rootDir, filePath)
	if err != nil {
		return "", err
	}

	exists := !symlinkInChain && fileExists(filePath)

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

	if err := os.WriteFile(filePath, content, mode); err != nil {
		return "", fmt.Errorf("writing %s: %w", filePath, err)
	}
	if mode != 0644 {
		if err := os.Chmod(filePath, mode); err != nil {
			return "", fmt.Errorf("chmod %s: %w", filePath, err)
		}
	}

	if exists {
		return fmt.Sprintf("UPDATED: %s", filePath), nil
	}
	return fmt.Sprintf("CREATED: %s", filePath), nil
}

func (i *Installer) checkOrCleanSymlinkChain(rootDir, targetPath string) (bool, error) {
	cleanRoot := filepath.Clean(rootDir)
	cleanTarget := filepath.Clean(targetPath)

	rel, err := filepath.Rel(cleanRoot, cleanTarget)
	if err != nil || strings.HasPrefix(rel, "..") {
		unlinked, uErr := UnlinkIfSymlink(cleanTarget, i.options.DryRun)
		return unlinked, uErr
	}

	cur := cleanRoot
	foundSymlink := false
	unlinkedRoot, err := UnlinkIfSymlink(cur, i.options.DryRun)
	if err != nil {
		return false, err
	}
	if unlinkedRoot {
		foundSymlink = true
		if i.options.DryRun {
			return true, nil
		}
	}

	if rel == "." {
		return foundSymlink, nil
	}

	parts := strings.Split(rel, string(os.PathSeparator))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		cur = filepath.Join(cur, part)
		unlinked, err := UnlinkIfSymlink(cur, i.options.DryRun)
		if err != nil {
			return false, err
		}
		if unlinked {
			foundSymlink = true
			if i.options.DryRun {
				return true, nil
			}
		}
	}
	return foundSymlink, nil
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
	inDescriptionBlock := false
	var descParts []string
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

		if inDescriptionBlock {
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				if trimmed != "" {
					descParts = append(descParts, trimmed)
				}
				continue
			}
			skill.Description = strings.Join(descParts, " ")
			inDescriptionBlock = false
		}

		if strings.HasPrefix(trimmed, "name:") {
			skill.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		} else if strings.HasPrefix(trimmed, "description:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			if val == ">-" || val == ">" || val == "|-" || val == "|" || val == "" {
				inDescriptionBlock = true
				descParts = nil
			} else {
				skill.Description = strings.Trim(val, `"'`)
			}
		} else if strings.HasPrefix(trimmed, "model:") {
			skill.Model = strings.TrimSpace(strings.TrimPrefix(trimmed, "model:"))
		} else if strings.HasPrefix(trimmed, "tags:") {
			skill.Tags = parseYAMLList(strings.TrimPrefix(trimmed, "tags:"))
		} else if strings.HasPrefix(trimmed, "languages:") {
			skill.Languages = parseYAMLList(strings.TrimPrefix(trimmed, "languages:"))
		}
		// Ignore unknown fields (allowed-tools, argument-hint, etc.)
	}
	if inDescriptionBlock {
		skill.Description = strings.Join(descParts, " ")
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

func MatchesFilter(skill Skill, tags, languages []string) bool {
	return matchesFilter(skill, tags, languages)
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
