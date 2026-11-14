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

	"github.com/futuregerald/futuregerald-claude-plugin/internal/backup"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/state"
)

const (
	HarnessClaude      = catalog.HarnessClaude
	HarnessAntigravity = catalog.HarnessAntigravity
	HarnessZed         = catalog.HarnessZed
	HarnessOpenCode    = catalog.HarnessOpenCode
)

func IsGeminiHarness(harness string) bool {
	return harness == HarnessAntigravity
}

func isAntigravityOnlySkill(skillName string) bool {
	return catalog.IsAntigravityOnlySkill(skillName)
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
	return AdaptMarkdownForHarness(content, harness, isGlobal)
}

func AdaptMarkdownForHarness(content []byte, harness string, isGlobal bool) []byte {
	if harness == "" || harness == HarnessClaude {
		return content
	}

	text := string(content)
	text = stripFrontmatterModelAndEffort(text)

	globalSkills := "~/.gemini/config/skills"
	globalSkillsHome := "$HOME/.gemini/config/skills"
	scopedSkills := ".agents/skills"
	globalAgents := "~/.gemini/config/plugins/antigravity-subagents/agents"
	globalAgentsHome := "$HOME/.gemini/config/plugins/antigravity-subagents/agents"
	scopedAgents := ".agents/agents"
	globalConfig := "~/.gemini/GEMINI.md"
	globalConfigHome := "$HOME/.gemini/GEMINI.md"
	scopedConfig := "GEMINI.md"
	povDir := "~/.gemini/point-of-view"
	povDirHome := "$HOME/.gemini/point-of-view"

	switch harness {
	case HarnessZed:
		globalSkills = "~/.agents/skills"
		globalSkillsHome = "$HOME/.agents/skills"
		scopedSkills = ".agents/skills"
		globalAgents = "~/.agents/skills"
		globalAgentsHome = "$HOME/.agents/skills"
		scopedAgents = ".agents/skills"
		globalConfig = "~/.config/zed/AGENTS.md"
		globalConfigHome = "$HOME/.config/zed/AGENTS.md"
		scopedConfig = "AGENTS.md"
		povDir = "~/.agents/point-of-view"
		povDirHome = "$HOME/.agents/point-of-view"
	case HarnessOpenCode:
		globalSkills = "~/.config/opencode/skills"
		globalSkillsHome = "$HOME/.config/opencode/skills"
		scopedSkills = ".opencode/skills"
		globalAgents = "~/.config/opencode/agents"
		globalAgentsHome = "$HOME/.config/opencode/agents"
		scopedAgents = ".opencode/agents"
		globalConfig = "~/.config/opencode/AGENTS.md"
		globalConfigHome = "$HOME/.config/opencode/AGENTS.md"
		scopedConfig = "AGENTS.md"
		povDir = "~/.config/opencode/point-of-view"
		povDirHome = "$HOME/.config/opencode/point-of-view"
	}

	if isGlobal {
		scopedSkills = globalSkills
		scopedAgents = globalAgents
	}

	pairs := []string{
		"$HOME/.claude/point-of-view", povDirHome,
		"~/.claude/point-of-view", povDir,
		"$HOME/.claude/skills", globalSkillsHome,
		"~/.claude/skills", globalSkills,
		"$HOME/.claude/agents", globalAgentsHome,
		"~/.claude/agents", globalAgents,
		"$HOME/.claude/CLAUDE.md", globalConfigHome,
		"~/.claude/CLAUDE.md", globalConfig,
		".claude/CLAUDE.md", scopedConfig,
		".claude/skills", scopedSkills,
		".claude/agents", scopedAgents,
	}
	if harness == HarnessAntigravity {
		pairs = append(pairs, "TaskCreate", "task.md")
	}

	replacer := strings.NewReplacer(pairs...)
	text = replacer.Replace(text)

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
		if strings.Contains(line, scopedConfig) {
			continue
		}
		lines[idx] = replaceStandaloneClaudeMDWith(line, scopedConfig)
	}

	return []byte(strings.Join(lines, "\n"))
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
	return replaceStandaloneClaudeMDWith(line, "GEMINI.md")
}

func replaceStandaloneClaudeMDWith(line, replacement string) string {
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
			b.WriteString(replacement)
			i = endPos
		} else {
			b.WriteString(line[i : absPos+len(needle)])
			i = endPos
		}
	}
	return b.String()
}

type Skill struct {
	Name        string
	Description string
	Model       string
	Tags        []string
	Languages   []string
	DirPath     string
	FilePath    string
	Content     []byte
}

type Options struct {
	Force  bool
	DryRun bool
}

type AgentNameFunc func(originalName string) string

type Installer struct {
	fsys    fs.FS
	options Options
}

func New(fsys fs.FS, opts Options) *Installer {
	return &Installer{
		fsys:    fsys,
		options: opts,
	}
}

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
	if strings.HasPrefix(p, "antigravity/skills/") {
		return strings.TrimPrefix(p, "antigravity/skills/")
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

func (i *Installer) tryParseSkillDir(dirPath string) (Skill, bool) {
	skillMD := path.Join(dirPath, "SKILL.md")
	content, err := fs.ReadFile(i.fsys, skillMD)
	if err != nil {
		skillMD = path.Join(dirPath, "skill.md")
		content, err = fs.ReadFile(i.fsys, skillMD)
		if err != nil {
			return Skill{}, false
		}
	}

	skill, err := parseSkill(content)
	if err != nil {
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

func (i *Installer) InstallSkills(destDir string, tags, languages []string) ([]string, error) {
	return i.InstallSkillsForHarness(destDir, HarnessClaude, false, tags, languages)
}

func (i *Installer) InstallSkillsForHarness(destDir, harness string, isGlobal bool, tags, languages []string) ([]string, error) {
	return i.InstallSelectedSkillsForHarness(destDir, harness, isGlobal, nil, tags, languages)
}

func (i *Installer) InstallSelectedSkillsForHarness(destDir, harness string, isGlobal bool, allowlist, tags, languages []string) ([]string, error) {
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

	allowMap := make(map[string]bool, len(allowlist))
	for _, s := range allowlist {
		allowMap[s] = true
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
	if harness == HarnessAntigravity {
		if _, statErr := fs.Stat(i.fsys, "antigravity/skills"); statErr == nil {
			overSkills, overErr := i.discoverSkillsInDir("antigravity/skills")
			if overErr == nil {
				for _, override := range overSkills {
					dirName := path.Base(override.DirPath)
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
	}

	for _, dirName := range orderedNames {
		if harness != HarnessAntigravity && isAntigravityOnlySkill(dirName) {
			continue
		}
		if len(allowMap) > 0 && !allowMap[dirName] {
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
				var fileContent []byte
				if !i.options.DryRun {
					fileContent, err = fs.ReadFile(i.fsys, file)
					if err != nil {
						return nil, fmt.Errorf("reading %s: %w", file, err)
					}
					if harness != HarnessClaude && strings.HasSuffix(strings.ToLower(relPath), ".md") {
						fileContent = AdaptMarkdownForHarness(fileContent, harness, isGlobal)
					}
				}
				targetPath := filepath.Join(destDir, filepath.FromSlash(relPath))
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
				var fileContent []byte
				if !i.options.DryRun {
					fileContent, err = fs.ReadFile(i.fsys, file)
					if err != nil {
						return nil, fmt.Errorf("reading %s: %w", file, err)
					}
				}
				targetPath := filepath.Join(destDir, filepath.FromSlash(relPath))
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

func (i *Installer) InstallAgents(destDir string, nameFunc AgentNameFunc) ([]string, error) {
	return i.InstallAgentsForHarness(destDir, HarnessClaude, false, nameFunc)
}

func (i *Installer) InstallAgentsForHarness(destDir, harness string, isGlobal bool, nameFunc AgentNameFunc) ([]string, error) {
	var results []string
	if destDir == "" || harness == HarnessZed {
		return results, nil
	}

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
	if harness == HarnessAntigravity {
		agentsDir = "antigravity/plugins/antigravity-subagents/agents"
	} else if harness == HarnessOpenCode {
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

type GlobalAssetOptions struct {
	LightweightConfig bool
	InstallHooks      bool
	SkipConfigFile    bool
	Env               platform.Env
}

func (i *Installer) InstallGlobalHarnessAssets(geminiHome, harness string) ([]string, error) {
	res, _, _, err := i.InstallGlobalHarnessAssetsWithOptions(geminiHome, harness, GlobalAssetOptions{
		InstallHooks: true,
		Env:          platform.Env{GOOS: "darwin"},
	})
	return res, err
}

func (i *Installer) InstallGlobalHarnessAssetsWithOptions(geminiHome, harness string, opts GlobalAssetOptions) ([]string, bool, bool, error) {
	if harness != HarnessAntigravity {
		return nil, false, false, nil
	}

	var results []string
	if info, err := os.Lstat(geminiHome); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, false, false, fmt.Errorf("refusing to modify symlinked gemini home directory %s", geminiHome)
	}
	configDir := filepath.Join(geminiHome, "config")
	if info, err := os.Lstat(configDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, false, false, fmt.Errorf("refusing to modify symlinked config directory %s", configDir)
	}

	if !opts.SkipConfigFile {
		geminiTemplate := "antigravity/GEMINI.md"
		if opts.LightweightConfig {
			if _, err := fs.Stat(i.fsys, "antigravity/GEMINI-LIGHT.md"); err == nil {
				geminiTemplate = "antigravity/GEMINI-LIGHT.md"
			}
		}
		if geminiMD, err := fs.ReadFile(i.fsys, geminiTemplate); err == nil {
			targetMD := filepath.Join(geminiHome, "GEMINI.md")
			res, err := i.writeFileWithRoot(targetMD, targetMD, geminiMD, 0644)
			if err != nil {
				return nil, false, false, err
			}
			results = append(results, res)
		}
	}

	if orgEx, err := fs.ReadFile(i.fsys, "antigravity/rules/org.example.md"); err == nil {
		rulesDir := filepath.Join(configDir, "rules")
		if info, err := os.Lstat(rulesDir); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, false, false, fmt.Errorf("refusing to modify symlinked rules directory %s", rulesDir)
		}
		targetOrg := filepath.Join(rulesDir, "org.example.md")
		res, err := i.writeFileWithRoot(targetOrg, targetOrg, orgEx, 0644)
		if err != nil {
			return nil, false, false, err
		}
		results = append(results, res)
	}

	hooksDest := filepath.Join(configDir, "hooks")
	hooksJSONDest := filepath.Join(configDir, "hooks.json")

	if !opts.InstallHooks {
		if !i.options.DryRun {
			_, _ = state.RemoveCircuitBreakerHook(hooksJSONDest, hooksDest, false)
		}
		return results, false, false, nil
	}

	unlinkedHooks, err := UnlinkIfSymlink(hooksDest, i.options.DryRun)
	if err != nil {
		return nil, false, false, err
	}
	if unlinkedHooks && i.options.DryRun {
		results = append(results, fmt.Sprintf("WOULD UNLINK SYMLINK: %s", hooksDest))
	}

	if _, err := fs.Stat(i.fsys, "antigravity/hooks"); err == nil {
		hookFiles, err := i.listDirFiles("antigravity/hooks")
		if err != nil {
			return nil, false, false, fmt.Errorf("listing antigravity/hooks: %w", err)
		}
		for _, hf := range hookFiles {
			base := path.Base(hf)
			if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") {
				continue
			}
			rel := strings.TrimPrefix(hf, "antigravity/hooks/")
			data, err := fs.ReadFile(i.fsys, hf)
			if err != nil {
				return nil, false, false, fmt.Errorf("reading %s: %w", hf, err)
			}
			mode := os.FileMode(0644)
			if strings.HasSuffix(base, ".sh") {
				mode = 0755
			}
			targetPath := filepath.Join(hooksDest, filepath.FromSlash(rel))
			res, err := i.writeFileWithRoot(hooksDest, targetPath, data, mode)
			if err != nil {
				return nil, false, false, err
			}
			results = append(results, res)
		}
	}

	hooksMerged := false
	hooksCreated := false
	if hooksJSONBytes, err := fs.ReadFile(i.fsys, "antigravity/hooks.json"); err == nil {
		if opts.Env.GOOS == "windows" {
			winCmd := platform.WindowsPythonHookCommand(geminiHome, nil)
			hooksJSONBytes = rewriteHooksJSONCommand(hooksJSONBytes, winCmd)
		}
		existedBefore := fileExists(hooksJSONDest)
		res, err := i.mergeHooksJSON(hooksJSONDest, hooksJSONBytes)
		if err != nil {
			return nil, false, false, err
		}
		results = append(results, res)
		hooksMerged = true
		hooksCreated = !existedBefore && strings.HasPrefix(res, "CREATED:")
	}

	return results, hooksMerged, hooksCreated, nil
}

func rewriteHooksJSONCommand(srcBytes []byte, newCommand string) []byte {
	replaced := strings.ReplaceAll(
		string(srcBytes),
		`"bash ~/.gemini/config/hooks/autonomous-breaker.sh"`,
		marshalJSONString(newCommand),
	)
	return []byte(replaced)
}

func marshalJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `"` + s + `"`
	}
	return string(b)
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

func (i *Installer) InstallCommands(destDir string) ([]string, error) {
	return i.InstallCommandsForHarness(destDir, HarnessClaude, nil)
}

func (i *Installer) InstallCommandsForHarness(destDir, harness string, allowedRelPaths []string) ([]string, error) {
	var results []string
	if destDir == "" || (harness != HarnessClaude && harness != HarnessOpenCode) {
		return results, nil
	}

	unlinked, err := UnlinkIfSymlink(destDir, i.options.DryRun)
	if err != nil {
		return nil, err
	}
	if unlinked && i.options.DryRun {
		results = append(results, fmt.Sprintf("WOULD UNLINK SYMLINK: %s", destDir))
	}

	commandsRoot := path.Join(harness, "commands")
	if _, err := fs.Stat(i.fsys, commandsRoot); err != nil {
		commandsRoot = "commands"
	}
	files, err := i.listDirFiles(commandsRoot)
	if err != nil {
		return results, nil
	}

	allowSet := make(map[string]bool, len(allowedRelPaths))
	for _, rel := range allowedRelPaths {
		allowSet[rel] = true
	}

	for _, file := range files {
		relPath := strings.TrimPrefix(file, commandsRoot+"/")
		if len(allowSet) > 0 && !allowSet[relPath] {
			continue
		}
		targetPath := filepath.Join(destDir, filepath.FromSlash(relPath))

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

type InstallPlan struct {
	Env                   platform.Env
	Catalog               *catalog.Catalog
	Harness               string
	Scope                 string
	ProjectDir            string
	TrackID               string
	CustomSkills          []string
	Tags                  []string
	Languages             []string
	Mode                  string
	SkipConfig            bool
	SkipAgents            bool
	SkipCommands          bool
	Backup                bool
	Force                 bool
	ForceAgents           bool
	ForceConfig           bool
	DryRun                bool
	Version               string
	CommitSHA             string
	ProjectConfigRenderer func(baseTemplate []byte, harness string, projectDir string) []byte
}

type InstallResult struct {
	Harness      string
	Scope        string
	TrackID      string
	Paths        platform.HarnessPaths
	Skills       []string
	Actions      []string
	WrittenFiles []string
	BackupID     string
	Receipt      state.Receipt
}

func extractWrittenPath(action string) (string, bool) {
	for _, prefix := range []string{"CREATED: ", "UPDATED: "} {
		if strings.HasPrefix(action, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(action, prefix)), true
		}
	}
	return "", false
}

func (i *Installer) ExecutePlan(plan InstallPlan) (*InstallResult, error) {
	cat := plan.Catalog
	if cat == nil {
		loaded, err := catalog.Load(i.fsys)
		if err != nil {
			return nil, err
		}
		cat = loaded
	}

	scope := plan.Scope
	if scope == "" {
		scope = "project"
	}
	projDir := plan.ProjectDir
	if projDir == "" {
		projDir = "."
	}

	paths, err := plan.Env.ResolvePaths(plan.Harness, scope, projDir)
	if err != nil {
		return nil, err
	}

	trackID := plan.TrackID
	if trackID == "" {
		trackID = "opinionated"
	}
	track, hasTrack := cat.TrackByID(trackID)
	if !hasTrack {
		return nil, fmt.Errorf("unknown track %q", trackID)
	}

	var selectedSkills []string
	if len(plan.CustomSkills) > 0 || trackID == "custom" {
		seen := make(map[string]bool)
		for _, s := range plan.CustomSkills {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if isAntigravityOnlySkill(s) && plan.Harness != HarnessAntigravity {
				continue
			}
			if !seen[s] {
				seen[s] = true
				selectedSkills = append(selectedSkills, s)
			}
		}
	} else {
		selectedSkills, err = cat.TrackSkills(trackID, plan.Harness)
		if err != nil {
			return nil, err
		}
	}

	mode := plan.Mode
	if mode == "" {
		mode = "full"
	}

	store := state.NewStore(plan.Env.StateFilePath())
	priorReceipt, hasPrior, _ := store.Find(plan.Harness, scope, paths.RootDir)
	if !hasPrior {
		if leg, ok := state.DiscoverLegacyReceipt(plan.Env, cat, plan.Harness, scope, projDir); ok {
			priorReceipt = leg
			hasPrior = true
		}
	}

	dryInst := New(i.fsys, Options{Force: true, DryRun: true})
	execInst := New(i.fsys, Options{Force: i.options.Force || plan.Force, DryRun: i.options.DryRun || plan.DryRun})
	var candidateTargets []string
	plannedSet := make(map[string]bool)
	addPlanned := func(p string) {
		clean := filepath.Clean(strings.TrimSpace(p))
		if clean != "" {
			plannedSet[clean] = true
			candidateTargets = append(candidateTargets, clean)
		}
	}
	if hasPrior && priorReceipt != nil {
		candidateTargets = append(candidateTargets, priorReceipt.InstalledFiles...)
	}
	if (mode == "config-only" || (mode == "full" && !plan.SkipConfig)) && paths.ConfigFile != "" {
		addPlanned(paths.ConfigFile)
	}
	if paths.HooksJSON != "" {
		candidateTargets = append(candidateTargets, paths.HooksJSON)
	}
	if mode == "full" && paths.SkillsDir != "" {
		if sim, sErr := dryInst.InstallSelectedSkillsForHarness(paths.SkillsDir, plan.Harness, scope == "global", selectedSkills, plan.Tags, plan.Languages); sErr == nil {
			for _, line := range sim {
				for _, pfx := range []string{"WOULD CREATE: ", "WOULD OVERWRITE: "} {
					if strings.HasPrefix(line, pfx) {
						addPlanned(strings.TrimPrefix(line, pfx))
					}
				}
			}
		}
	}
	if (mode == "agents-only" || (mode == "full" && track.InstallAgents && !plan.SkipAgents)) && paths.AgentsDir != "" {
		if sim, aErr := dryInst.InstallAgentsForHarness(paths.AgentsDir, plan.Harness, scope == "global", nil); aErr == nil {
			for _, line := range sim {
				for _, pfx := range []string{"WOULD CREATE: ", "WOULD OVERWRITE: "} {
					if strings.HasPrefix(line, pfx) {
						addPlanned(strings.TrimPrefix(line, pfx))
					}
				}
			}
		}
	}
	if mode == "full" && !plan.SkipCommands && paths.CommandsDir != "" {
		allowedCmds := cat.CommandsForSkills(plan.Harness, selectedSkills)
		if sim, cErr := dryInst.InstallCommandsForHarness(paths.CommandsDir, plan.Harness, allowedCmds); cErr == nil {
			for _, line := range sim {
				for _, pfx := range []string{"WOULD CREATE: ", "WOULD OVERWRITE: "} {
					if strings.HasPrefix(line, pfx) {
						addPlanned(strings.TrimPrefix(line, pfx))
					}
				}
			}
		}
	}
	if scope == "global" && plan.Harness == HarnessAntigravity && (mode == "full" || (mode == "config-only" && !plan.SkipConfig)) {
		if sim, _, _, gErr := dryInst.InstallGlobalHarnessAssetsWithOptions(paths.GeminiHome, plan.Harness, GlobalAssetOptions{
			LightweightConfig: track.LightweightConfig,
			InstallHooks:      track.InstallHooks,
			SkipConfigFile:    plan.SkipConfig,
			Env:               plan.Env,
		}); gErr == nil {
			for _, line := range sim {
				for _, pfx := range []string{"WOULD CREATE: ", "WOULD OVERWRITE: "} {
					if strings.HasPrefix(line, pfx) {
						addPlanned(strings.TrimPrefix(line, pfx))
					}
				}
			}
		}
	}

	var backupID string
	if plan.Backup && !execInst.options.DryRun {
		mgr := backup.NewManager(plan.Env.BackupsDir())
		mf, bErr := mgr.CreateSnapshotWithRoot(plan.Harness, scope, paths.RootDir, trackID, candidateTargets, plan.Env.StateFilePath())
		if bErr != nil {
			return nil, fmt.Errorf("creating safety backup: %w", bErr)
		}
		backupID = mf.ID
	}

	var actions []string
	var writtenFiles []string
	recordActions := func(lines []string) {
		for _, l := range lines {
			actions = append(actions, l)
			if p, ok := extractWrittenPath(l); ok {
				writtenFiles = append(writtenFiles, filepath.Clean(p))
			}
		}
	}

	if mode == "full" && paths.SkillsDir != "" {
		res, sErr := execInst.InstallSelectedSkillsForHarness(paths.SkillsDir, plan.Harness, scope == "global", selectedSkills, plan.Tags, plan.Languages)
		if sErr != nil {
			return nil, sErr
		}
		recordActions(res)
	}

	if (mode == "agents-only" || (mode == "full" && track.InstallAgents && !plan.SkipAgents)) && paths.AgentsDir != "" {
		agentInst := execInst
		if plan.ForceAgents && !execInst.options.Force {
			agentInst = New(i.fsys, Options{Force: true, DryRun: execInst.options.DryRun})
		}
		res, aErr := agentInst.InstallAgentsForHarness(paths.AgentsDir, plan.Harness, scope == "global", nil)
		if aErr != nil {
			return nil, aErr
		}
		recordActions(res)
	}

	if mode == "full" && !plan.SkipCommands && paths.CommandsDir != "" {
		allowedCmds := cat.CommandsForSkills(plan.Harness, selectedSkills)
		res, cErr := execInst.InstallCommandsForHarness(paths.CommandsDir, plan.Harness, allowedCmds)
		if cErr != nil {
			return nil, cErr
		}
		recordActions(res)
	}

	hooksMerged := false
	hooksCreated := false
	if scope == "global" && plan.Harness == HarnessAntigravity && (mode == "full" || (mode == "config-only" && !plan.SkipConfig)) {
		cfgInst := execInst
		if plan.ForceConfig && !execInst.options.Force {
			cfgInst = New(i.fsys, Options{Force: true, DryRun: execInst.options.DryRun})
		}
		res, hm, hc, gErr := cfgInst.InstallGlobalHarnessAssetsWithOptions(paths.GeminiHome, plan.Harness, GlobalAssetOptions{
			LightweightConfig: track.LightweightConfig,
			InstallHooks:      track.InstallHooks,
			SkipConfigFile:    plan.SkipConfig,
			Env:               plan.Env,
		})
		if gErr != nil {
			return nil, gErr
		}
		recordActions(res)
		hooksMerged = hm
		hooksCreated = hc
	} else if (mode == "config-only" || (mode == "full" && !plan.SkipConfig)) && paths.ConfigFile != "" {
		cfgInst := execInst
		if plan.ForceConfig && !execInst.options.Force {
			cfgInst = New(i.fsys, Options{Force: true, DryRun: execInst.options.DryRun})
		}
		if scope == "global" {
			tpl, tErr := cat.GlobalConfigTemplate(plan.Harness, track.LightweightConfig)
			if tErr != nil {
				return nil, tErr
			}
			res, wErr := cfgInst.writeFileWithRoot(paths.ConfigFile, paths.ConfigFile, tpl, 0644)
			if wErr != nil {
				return nil, wErr
			}
			recordActions([]string{res})
		} else {
			tpl, tErr := cat.ProjectConfigTemplate(plan.Harness)
			if tErr != nil {
				return nil, tErr
			}
			if plan.ProjectConfigRenderer != nil {
				tpl = plan.ProjectConfigRenderer(tpl, plan.Harness, projDir)
			}
			res, wErr := cfgInst.writeFileWithRoot(paths.ConfigFile, paths.ConfigFile, tpl, 0644)
			if wErr != nil {
				return nil, wErr
			}
			recordActions([]string{res})
		}
	}

	if hasPrior && priorReceipt != nil && mode == "full" && !execInst.options.DryRun {
		cleanHooksJSON := ""
		if paths.HooksJSON != "" {
			cleanHooksJSON = filepath.Clean(paths.HooksJSON)
		}
		cleanConfig := ""
		if paths.ConfigFile != "" {
			cleanConfig = filepath.Clean(paths.ConfigFile)
		}
		cleanAgentsPrefix := ""
		if paths.AgentsDir != "" {
			cleanAgentsPrefix = filepath.Clean(paths.AgentsDir) + string(os.PathSeparator)
		}
		cleanCommandsPrefix := ""
		if paths.CommandsDir != "" {
			cleanCommandsPrefix = filepath.Clean(paths.CommandsDir) + string(os.PathSeparator)
		}
		for _, oldFile := range priorReceipt.InstalledFiles {
			cleanOld := filepath.Clean(oldFile)
			if plannedSet[cleanOld] || isPreservedLocalFile(cleanOld) {
				continue
			}
			if cleanOld == cleanHooksJSON || cleanOld == cleanConfig {
				continue
			}
			if plan.SkipAgents && cleanAgentsPrefix != "" && strings.HasPrefix(cleanOld, cleanAgentsPrefix) {
				continue
			}
			if plan.SkipCommands && cleanCommandsPrefix != "" && strings.HasPrefix(cleanOld, cleanCommandsPrefix) {
				continue
			}
			if err := backup.RejectSymlinkInChain(cleanOld); err != nil {
				continue
			}
			if _, err := os.Lstat(cleanOld); err == nil {
				if err := os.Remove(cleanOld); err == nil {
					actions = append(actions, fmt.Sprintf("REMOVED: %s", cleanOld))
					backup.PruneEmptyParents(filepath.Dir(cleanOld), 3)
				}
			}
		}
	}

	allTracked := make(map[string]bool)
	for _, wf := range writtenFiles {
		if paths.HooksJSON != "" && filepath.Clean(wf) == filepath.Clean(paths.HooksJSON) {
			continue
		}
		allTracked[filepath.Clean(wf)] = true
	}
	if hasPrior && priorReceipt != nil {
		for _, of := range priorReceipt.InstalledFiles {
			cleanOld := filepath.Clean(of)
			if paths.HooksJSON != "" && cleanOld == filepath.Clean(paths.HooksJSON) {
				continue
			}
			if _, err := os.Lstat(cleanOld); err == nil {
				allTracked[cleanOld] = true
			}
		}
	}

	var trackedList []string
	for f := range allTracked {
		trackedList = append(trackedList, f)
	}
	sort.Strings(trackedList)

	receipt := state.Receipt{
		ID:               state.ReceiptKey(plan.Harness, scope, paths.RootDir),
		Harness:          plan.Harness,
		Scope:            scope,
		RootDir:          filepath.Clean(paths.RootDir),
		Track:            trackID,
		Skills:           selectedSkills,
		InstalledFiles:   trackedList,
		HooksMerged:      hooksMerged,
		HooksFileCreated: hooksCreated,
		Version:          plan.Version,
		CommitSHA:        plan.CommitSHA,
		LastBackupID:     backupID,
	}
	if hasPrior && priorReceipt != nil {
		if backupID == "" {
			receipt.LastBackupID = priorReceipt.LastBackupID
		}
		if receipt.CommitSHA == "" {
			receipt.CommitSHA = priorReceipt.CommitSHA
		}
		if priorReceipt.HooksFileCreated && hooksMerged {
			receipt.HooksFileCreated = true
		}
	}

	if !execInst.options.DryRun {
		_ = store.Upsert(receipt)
	}

	return &InstallResult{
		Harness:      plan.Harness,
		Scope:        scope,
		TrackID:      trackID,
		Paths:        paths,
		Skills:       selectedSkills,
		Actions:      actions,
		WrittenFiles: trackedList,
		BackupID:     backupID,
		Receipt:      receipt,
	}, nil
}

func (i *Installer) ListSkills() ([]Skill, error) {
	return i.discoverSkills()
}

func (i *Installer) ListAllSkills() ([]Skill, error) {
	return i.discoverSkills()
}

func (i *Installer) ListSkillsForHarness(harness string) ([]Skill, error) {
	baseSkills, err := i.discoverSkills()
	if err != nil {
		return nil, err
	}
	if harness != HarnessAntigravity {
		var filtered []Skill
		for _, s := range baseSkills {
			if isAntigravityOnlySkill(s.Name) {
				continue
			}
			filtered = append(filtered, s)
		}
		return filtered, nil
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
		merged = append(merged, byDir[d])
	}
	return merged, nil
}

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
	const (
		maxFileBytes  = 10 * 1024 * 1024
		maxTotalBytes = 100 * 1024 * 1024
	)
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	cleanDest := filepath.Clean(destDir)
	var totalBytes int64
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeXGlobalHeader || header.Typeflag == tar.TypeXHeader || header.Name == "pax_global_header" {
			continue
		}

		target := filepath.Join(cleanDest, header.Name)
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("illegal path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := backup.RejectSymlinkInChain(cleanTarget); err != nil {
				return err
			}
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size > maxFileBytes {
				return fmt.Errorf("archive entry %s exceeds 10 MiB limit (%d bytes)", header.Name, header.Size)
			}
			if err := backup.RejectSymlinkInChain(cleanTarget); err != nil {
				return err
			}
			if info, lErr := os.Lstat(cleanTarget); lErr == nil && info.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(cleanTarget); err != nil {
					return err
				}
			}
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
				return err
			}
			mode := os.FileMode(0644)
			if header.Mode&0111 != 0 {
				mode = 0755
			}
			f, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			n, copyErr := io.Copy(f, io.LimitReader(tr, maxFileBytes+1))
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			if n > maxFileBytes {
				return fmt.Errorf("archive entry %s exceeds 10 MiB limit", header.Name)
			}
			totalBytes += n
			if totalBytes > maxTotalBytes {
				return fmt.Errorf("archive total extracted size exceeds 100 MiB limit")
			}
		default:
			return fmt.Errorf("unsupported tar entry type %d for %s", header.Typeflag, header.Name)
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
	}
	if inDescriptionBlock {
		skill.Description = strings.Join(descParts, " ")
	}

	if skill.Name == "" {
		return skill, fmt.Errorf("skill missing name in frontmatter")
	}
	return skill, nil
}

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
