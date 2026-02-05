# Merge skill-installer CLI into futuregerald-claude-plugin

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add the skill-installer CLI tool into the futuregerald-claude-plugin repo so users can install the plugin via Claude Code natively OR use the CLI to install the plugin's skills, agents, commands, and project config for Claude Code, Cursor, GitHub Copilot, OpenCode, and VS Code.

**Architecture:** The plugin repo is the single source of truth for all content (skills, agents, templates, commands). The Go CLI is added purely as installation infrastructure - it reads from the plugin's existing directory structure and installs to various targets. No content is brought over from the skill-installer repo - only the Go code that handles multi-framework installation. `//go:embed` embeds the plugin's skills, agents, templates, and commands at build time. The CLI uses `fs.FS` interface (not concrete `embed.FS`) for testability, and `path.Join` (not `filepath.Join`) for all embedded FS path operations.

**Tech Stack:** Go 1.21+, Cobra CLI, io/fs, embed.FS, YAML frontmatter

---

## Source of Truth

**The futuregerald-claude-plugin content is canonical.** The CLI is just a delivery mechanism. Specifically:

- **Skills**: The 32 skills in `skills/` (directory-based, SKILL.md + references/templates)
- **Agents**: The 6 agents in `agents/`
- **Templates**: CLAUDE-BASE.md + language templates in `templates/`
- **Commands**: `/init-claude-md` in `commands/`

Nothing from the skill-installer's `skills/`, `agents/`, or `templates/` directories is brought over. Only the Go source code (`main.go`, `internal/`, `go.mod`, `go.sum`) that powers the CLI is imported, and it will be rewritten to work with the plugin's content structure.

---

## Target Framework Reference

| Target | Skills Path (project) | Skills Path (global) | Agents Path | Config File | Source |
|--------|----------------------|---------------------|-------------|-------------|--------|
| Claude Code | `.claude/skills/` | `~/.claude/skills/` | `.claude/agents/` | `CLAUDE.md` | Native |
| GitHub Copilot | `.github/skills/` | `~/.copilot/skills/` | `.github/*.agent.md` | `.github/copilot-instructions.md` | [docs](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills) |
| Cursor | `.cursor/skills/` | — | `.cursor/agents/` | `.cursorrules` | Cursor docs |
| OpenCode | `.opencode/skills/` | — | `.opencode/agents/` | — | OpenCode docs |
| VS Code | `.vscode/claude/skills/` | — | `.vscode/claude/agents/` | — | VS Code docs |

Note: Copilot and Claude Code support both project-scoped and global installation. The CLI will prompt users to choose.

---

## Packs Concept: Replaced by Tags

The original installer had a "pack" concept (core, go, python, typescript, rust) based on nested directories. The plugin's skills are all flat top-level directories under `skills/` — no nesting.

**Decision:** Drop the `--pack` flag and `ListPacks()`. Replace with tag-based filtering using the existing `--tag` flag. Skills can be tagged in their SKILL.md frontmatter (e.g., `tags: [workflow, core]` or `tags: [framework, adonisjs]`). This is simpler, more flexible, and doesn't require restructuring the skill directories.

To make this work, existing plugin skills that lack `tags` in their frontmatter will need tags added. This is a content enhancement, not a structural change.

---

## Target Directory Structure (After Merge)

```
futuregerald-claude-plugin/
├── .claude-plugin/
│   └── plugin.json                  # Claude Code plugin manifest (unchanged)
├── .gitignore                       # NEW - Go build artifacts, skill-zips
│
├── main.go                          # Go CLI entry point (adapted from installer)
├── go.mod                           # Go module (new module path)
├── go.sum                           # Go dependencies
├── Makefile                         # Build targets for CLI binary
├── internal/                        # Go internal packages (adapted from installer)
│   ├── config/
│   │   └── config.go               # Config loading (simplified)
│   └── installer/
│       ├── installer.go             # Rewritten for directory-based skills
│       └── installer_test.go        # Rewritten tests
│
├── skills/                          # ALL skills (UNCHANGED except case fixes)
│   ├── systematic-debugging/
│   │   ├── SKILL.md
│   │   ├── defense-in-depth.md
│   │   └── ...
│   ├── code-simplifier/
│   │   └── SKILL.md                 # RENAMED from code-simplifier.md
│   ├── design-principles/
│   │   └── SKILL.md                 # RENAMED from skill.md
│   ├── turso-best-practices/
│   │   └── SKILL.md                 # RENAMED from skill.md
│   └── ... (all other skills unchanged)
│
├── agents/                          # ALL agents (UNCHANGED)
│   ├── code-quality-reviewer.md
│   ├── code-simplifier.md
│   ├── codebase-searcher.md
│   ├── debugger.md
│   ├── implementer.md
│   └── spec-reviewer.md
│
├── commands/                        # Claude Code commands (UNCHANGED)
│   └── init-claude-md/
│       └── COMMAND.md
│
├── templates/                       # All templates (UNCHANGED)
│   ├── CLAUDE-BASE.md
│   └── languages/
│       ├── svelte.md, react.md, nodejs.md, adonisjs.md
│       ├── go.md, python.md, ruby.md, rust.md, php.md
│
├── README.md                        # Updated for both install methods
└── LICENSE
```

---

## Task 1: Initialize repo and bring in Go CLI infrastructure

**Files:**
- Create: `main.go` (from skill-installer, will be adapted)
- Create: `go.mod` (new module path)
- Create: `go.sum` (from skill-installer)
- Create: `internal/config/config.go` (from skill-installer)
- Create: `internal/installer/installer.go` (from skill-installer, placeholder until Task 4)
- Create: `internal/installer/installer_test.go` (from skill-installer, placeholder until Task 7)
- Create: `Makefile`
- Create: `.gitignore`

**Step 1: Initialize git repo if needed**

```bash
cd /Users/geraldonyango/Documents/dev/futuregerald-claude-plugin
git init  # if not already a repo
```

**Step 2: Copy Go source files from skill-installer**

```bash
cp /Users/geraldonyango/Documents/dev/skill-installer/main.go .
cp /Users/geraldonyango/Documents/dev/skill-installer/go.mod .
cp /Users/geraldonyango/Documents/dev/skill-installer/go.sum .
cp -r /Users/geraldonyango/Documents/dev/skill-installer/internal .
```

**Step 3: Update go.mod module path**

Change:
```
module github.com/futuregerald/skill-installer
```
To:
```
module github.com/futuregerald/futuregerald-claude-plugin
```

**Step 4: Update all import paths in main.go**

Change:
```go
"github.com/futuregerald/skill-installer/internal/config"
"github.com/futuregerald/skill-installer/internal/installer"
```
To:
```go
"github.com/futuregerald/futuregerald-claude-plugin/internal/config"
"github.com/futuregerald/futuregerald-claude-plugin/internal/installer"
```

**Step 5: Create Makefile**

```makefile
BINARY_NAME=skill-installer
VERSION=2.0.0

.PHONY: build clean test install

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
```

Note: Uses `install -m 755` instead of `mv` so the binary stays in the build directory and gets proper permissions.

**Step 6: Create .gitignore**

```
# Go build artifacts
skill-installer
*.exe

# Generated archives
skill-zips/

# OS files
.DS_Store
```

**Step 7: Verify Go build compiles**

```bash
go build -o skill-installer .
```

Expected: Compiles (embed paths will fail at runtime until Task 3, but compilation should succeed).

**Step 8: Commit**

```bash
git add main.go go.mod go.sum internal/ Makefile .gitignore
git commit -m "chore: import Go CLI infrastructure from skill-installer"
```

---

## Task 2: Fix SKILL.md naming inconsistencies

Three skills have non-standard filenames that would be missed by the discovery function.

**Files:**
- Rename: `skills/code-simplifier/code-simplifier.md` → `skills/code-simplifier/SKILL.md`
- Rename: `skills/design-principles/skill.md` → `skills/design-principles/SKILL.md`
- Rename: `skills/turso-best-practices/skill.md` → `skills/turso-best-practices/SKILL.md`

**Step 1: Rename the files**

```bash
cd /Users/geraldonyango/Documents/dev/futuregerald-claude-plugin
git mv skills/code-simplifier/code-simplifier.md skills/code-simplifier/SKILL.md
git mv skills/design-principles/skill.md skills/design-principles/SKILL.md
git mv skills/turso-best-practices/skill.md skills/turso-best-practices/SKILL.md
```

**Step 2: Verify all 32 skills have SKILL.md**

```bash
find skills -maxdepth 2 -name "SKILL.md" | wc -l
```

Expected: `32`

**Step 3: Commit**

```bash
git commit -m "fix: normalize SKILL.md casing for all skills"
```

---

## Task 3: Update embed directives and main.go Target system

**Files:**
- Modify: `main.go`

**Step 1: Update the embed directive**

Replace:
```go
//go:embed skills/*.md skills/**/*.md templates/*.md agents/*.md
var content embed.FS
```
With:
```go
//go:embed all:skills all:agents all:templates all:commands
var content embed.FS
```

**Step 2: Add `"io/fs"` and `"path"` to imports, remove unused imports**

Add to imports:
```go
"io/fs"
"path"
```

These will be needed throughout main.go for embedded FS operations.

**Step 3: Update Target struct to include AgentsPath and scope options**

```go
type Target struct {
    Name        string
    SkillsPath  string // Project-scoped skills
    AgentsPath  string // Project-scoped agents
    CommandsPath string // Project-scoped commands (if supported)
    ConfigPath  string // Framework config file
    GlobalSkillsPath string // User-level skills (empty if not supported)
    GlobalAgentsPath string // User-level agents (empty if not supported)
}
```

**Step 4: Update targets map**

```go
var targets = map[string]Target{
    "claude": {
        Name:             "Claude Code",
        SkillsPath:       ".claude/skills",
        AgentsPath:       ".claude/agents",
        CommandsPath:     ".claude/commands",
        ConfigPath:       "CLAUDE.md",
        GlobalSkillsPath: filepath.Join(homeDir(), ".claude", "skills"),
        GlobalAgentsPath: filepath.Join(homeDir(), ".claude", "agents"),
    },
    "copilot": {
        Name:             "GitHub Copilot",
        SkillsPath:       ".github/skills",
        AgentsPath:       ".github",          // Copilot uses .github/*.agent.md
        CommandsPath:     "",                  // Not supported
        ConfigPath:       ".github/copilot-instructions.md",
        GlobalSkillsPath: filepath.Join(homeDir(), ".copilot", "skills"),
        GlobalAgentsPath: "",
    },
    "cursor": {
        Name:         "Cursor",
        SkillsPath:   ".cursor/skills",
        AgentsPath:   ".cursor/agents",
        CommandsPath: "",
        ConfigPath:   ".cursorrules",
    },
    "opencode": {
        Name:         "OpenCode",
        SkillsPath:   ".opencode/skills",
        AgentsPath:   ".opencode/agents",
        CommandsPath: "",
        ConfigPath:   "",
    },
    "vscode": {
        Name:         "VS Code (with Claude extension)",
        SkillsPath:   ".vscode/claude/skills",
        AgentsPath:   ".vscode/claude/agents",
        CommandsPath: "",
        ConfigPath:   "",
    },
}
```

Add a `homeDir()` helper:
```go
func homeDir() string {
    h, err := os.UserHomeDir()
    if err != nil {
        return ""
    }
    return h
}
```

**Step 5: Remove `--pack` and `--agents` flags**

Remove from rootCmd.Flags():
- `--pack` / `-p` (replaced by `--tag`)
- `--agents` / `-a` (agents install automatically)

Add new variables to the `var` block alongside `force`, `dryRun`, etc.:
```go
var (
    // ... existing vars ...
    skipAgents  bool
    skipCommands bool
    globalInstall bool
)
```

Add new flags:
- `--skip-agents` (bool, skip agent installation) → `&skipAgents`
- `--skip-commands` (bool, skip command installation) → `&skipCommands`
- `--global` (bool, install to global/user-level directory instead of project) → `&globalInstall`

```go
rootCmd.Flags().BoolVar(&skipAgents, "skip-agents", false, "Skip installing agents")
rootCmd.Flags().BoolVar(&skipCommands, "skip-commands", false, "Skip installing commands")
rootCmd.Flags().BoolVar(&globalInstall, "global", false, "Install to global/user-level directory")
```

**Step 6: Remove `getAgentsLocation` function** (no longer needed)

**Step 7: Replace `askUpdateClaudeMD` with generic `askUpdateConfig`**

```go
func askUpdateConfig(reader *bufio.Reader, target Target) (bool, error) {
    if target.ConfigPath == "" {
        return false, nil
    }
    if nonInteract {
        return true, nil
    }
    fmt.Printf("\nGenerate %s? [Y/n]: ", target.ConfigPath)
    input, err := reader.ReadString('\n')
    if err != nil {
        return false, err
    }
    input = strings.TrimSpace(strings.ToLower(input))
    return input == "" || input == "y" || input == "yes", nil
}
```

**Step 8: Add askScope function for targets that support global installation**

```go
func askScope(reader *bufio.Reader, target Target) (string, error) {
    // No global path available for this target
    if target.GlobalSkillsPath == "" {
        return "project", nil
    }

    // --global flag bypasses prompt
    if globalInstall {
        return "global", nil
    }

    // Non-interactive defaults to project
    if nonInteract {
        return "project", nil
    }

    fmt.Println("\nWhere should skills be installed?")
    fmt.Println("  1) Project-scoped (current directory)")
    fmt.Println("  2) Global (available to all projects)")
    fmt.Print("Enter choice [1]: ")

    input, err := reader.ReadString('\n')
    if err != nil {
        return "", err
    }
    input = strings.TrimSpace(input)

    if input == "2" {
        return "global", nil
    }
    return "project", nil
}
```

**Step 9: Add `"copilot"` to interactive target options**

In `getTarget()`, update the options slice:
```go
options := []string{"claude", "copilot", "cursor", "opencode", "vscode"}
```

**Step 10: Remove `updateClaudeMDFile` function** (replaced in Task 5)

**Step 11: Remove `runPacks` command** (packs concept removed)

Remove from rootCmd.AddCommand. Keep `list`, `init`, `version`.

**Step 12: Fix deprecated `strings.Title` usage in `runPacks` and anywhere else**

Since `runPacks` is removed, check for any remaining `strings.Title` calls. Replace with:
```go
func titleCase(s string) string {
    if s == "" {
        return s
    }
    return strings.ToUpper(s[:1]) + s[1:]
}
```

**Step 13: Update `runInit` to create directory-based skills**

Change from creating a flat `name.md` to creating `name/SKILL.md`:

```go
func runInit(cmd *cobra.Command, args []string) error {
    name := args[0]
    // ... (get flags as before)

    skillDir := name
    filename := filepath.Join(skillDir, "SKILL.md")

    if fileExists(filename) && !force {
        return fmt.Errorf("%s already exists (use --force to overwrite)", filename)
    }

    content := installer.GenerateSkillTemplate(name, desc, model, initTags, initLangs)

    if dryRun {
        fmt.Printf("WOULD CREATE: %s\n", filename)
        return nil
    }

    if err := os.MkdirAll(skillDir, 0755); err != nil {
        return fmt.Errorf("creating directory %s: %w", skillDir, err)
    }

    if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
        return fmt.Errorf("writing %s: %w", filename, err)
    }

    fmt.Printf("CREATED: %s\n", filename)
    fmt.Println("\nEdit the file to customize your skill, then move the directory to your skills location.")
    return nil
}
```

**Step 14: Verify compilation**

```bash
go build -o skill-installer .
```

Expected: Compiles successfully.

**Step 15: Commit**

```bash
git add main.go
git commit -m "refactor: update targets, embed directives, remove packs, add scope prompt"
```

---

## Task 4: Rewrite installer.go for directory-based skills

The installer must be rewritten to:
1. Use `fs.FS` interface (not `embed.FS`) for testability
2. Use `path.Join` (not `filepath.Join`) for all embedded FS paths
3. Use `fs.ReadDir()`, `fs.ReadFile()`, `fs.WalkDir()` free functions (not methods)
4. Walk `skills/` finding directories that contain `SKILL.md`
5. Copy entire skill directories (SKILL.md + references + templates)
6. Install agent `.md` files
7. Install command directories
8. Support tag-based filtering (no more pack system)

**Files:**
- Rewrite: `internal/installer/installer.go`

**Step 1: Update imports**

```go
package installer

import (
    "fmt"
    "io/fs"
    "net/http"
    "os"
    "os/exec"
    "path"           // For embedded FS paths (always forward slashes)
    "path/filepath"  // For OS filesystem paths only
    "strings"
    // Remove: "embed" (no longer needed in this file)
    // Remove: "archive/tar", "compress/gzip", "io" (keep if --from is retained)
)
```

**Step 2: Update Installer struct to use `fs.FS`**

```go
type Installer struct {
    fsys    fs.FS   // Changed from embed.FS to fs.FS
    options Options
}

func New(fsys fs.FS, opts Options) *Installer {
    return &Installer{
        fsys:    fsys,
        options: opts,
    }
}
```

**Step 3: Update Skill struct**

```go
type Skill struct {
    Name        string
    Description string
    Model       string
    Tags        []string
    Languages   []string
    DirPath     string   // Directory path within embedded FS (e.g., "skills/systematic-debugging")
    FilePath    string   // SKILL.md path within embedded FS
    Content     []byte   // Content of SKILL.md
}
```

Note: `Pack` field removed (packs concept dropped). `Files` field not needed (we walk at install time).

**Step 4: Write `discoverSkills` function**

```go
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
```

**Step 5: Write `listDirFiles` helper**

```go
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
```

**Step 6: Write `InstallSkills` - copies entire skill directories**

```go
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
    }

    return results, nil
}
```

Note: Uses `path.Join` / `strings.TrimPrefix` for embedded FS paths, `filepath.Join` for OS target paths.

**Step 7: Write `InstallAgents` - copies agent .md files with optional renaming**

The function accepts a `nameFunc` parameter to handle target-specific naming conventions (e.g., Copilot requires `*.agent.md`):

```go
// AgentNameFunc transforms an agent filename for the target framework.
// Pass nil to keep original names.
type AgentNameFunc func(originalName string) string

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

// CopilotAgentName converts "debugger.md" to "debugger.agent.md"
func CopilotAgentName(name string) string {
    return strings.TrimSuffix(name, ".md") + ".agent.md"
}
```
```

**Step 8: Write `InstallCommands` - copies command directories**

```go
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
```

**Step 9: Write `ListSkills` and `ListAllSkills`**

```go
func (i *Installer) ListSkills() ([]Skill, error) {
    return i.discoverSkills()
}

// ListAllSkills is an alias kept for backward compat
func (i *Installer) ListAllSkills() ([]Skill, error) {
    return i.discoverSkills()
}
```

**Step 10: Update `parseSkill` to handle plugin-style frontmatter**

The parser must tolerate:
- Missing `model` (default to "sonnet")
- Missing `tags` and `languages` (default to empty)
- Extra fields like `allowed-tools`, `argument-hint` (ignore)
- Missing `name` (return error, caller handles fallback)

The existing `parseSkill` function mostly works. Just make `name` not strictly required (return partial Skill with error, let caller decide):

```go
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
            // Handle multi-line description (just take first line for now)
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
```

**Step 11: Keep and adapt `InstallFromLocal`, `InstallFromGit`, `InstallFromURL`**

These support the `--from` flag for installing skills from external sources. Update `InstallFromLocal` to look for skill directories (containing SKILL.md) instead of flat .md files:

```go
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
```

Note: This copies ALL files (not just .md), preserving directory structure. This handles both flat and directory-based skills from external sources.

**Step 12: Fix zip-slip vulnerability in `extractTarGz`**

```go
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
```

**Step 13: Remove `InstallAgentsMD` function** (replaced by per-file InstallAgents)

**Step 14: Remove `ListPacks` function** (packs concept dropped)

**Step 15: Keep `writeFile`, `fileExists`, `matchesFilter`, `contains`, `parseYAMLList` functions** (mostly unchanged)

**Step 15a: Update `GenerateSkillTemplate` to replace deprecated `strings.Title`**

Add a `titleCase` helper to the installer package (since `main.go` can't share its helper across packages):

```go
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
```

Then in `GenerateSkillTemplate`, replace:
```go
strings.Title(strings.ReplaceAll(name, "-", " "))
```
With:
```go
titleCase(strings.ReplaceAll(name, "-", " "))
```

**Step 16: Commit**

```bash
git add internal/installer/installer.go
git commit -m "refactor: rewrite installer for directory-based skills with fs.FS interface"
```

---

## Task 5: Rewrite runInstall in main.go

**Files:**
- Modify: `main.go`

**Step 1: Rewrite `runInstall` function**

```go
func runInstall(cmd *cobra.Command, args []string) error {
    reader := bufio.NewReader(os.Stdin)

    // Load config
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("loading config: %w", err)
    }
    applyConfig(cfg)

    // Get target framework
    target, err := getTarget(reader)
    if err != nil {
        return err
    }

    // Ask about scope (project vs global) for targets that support it
    scope, err := askScope(reader, target)
    if err != nil {
        return err
    }

    // Determine installation paths based on scope
    var skillsDest, agentsDest, commandsDest string
    if scope == "global" {
        skillsDest = target.GlobalSkillsPath
        agentsDest = target.GlobalAgentsPath
    } else {
        skillsDest = filepath.Join(".", target.SkillsPath)
        agentsDest = filepath.Join(".", target.AgentsPath)
        commandsDest = filepath.Join(".", target.CommandsPath)
    }

    // Ask about config file generation (project-scoped only)
    updateConfig := false
    if scope == "project" && target.ConfigPath != "" && !skipClaude {
        updateConfig, err = askUpdateConfig(reader, target)
        if err != nil {
            return err
        }
    }

    inst := installer.New(content, installer.Options{
        Force:  force,
        DryRun: dryRun,
    })

    // Install skills
    fmt.Println("\nInstalling skills...")
    var results []string

    if fromSource != "" {
        if strings.HasPrefix(fromSource, "http://") || strings.HasPrefix(fromSource, "https://") {
            if strings.Contains(fromSource, "github.com") || strings.Contains(fromSource, "gitlab.com") {
                results, err = inst.InstallFromGit(fromSource, skillsDest)
            } else {
                results, err = inst.InstallFromURL(fromSource, skillsDest)
            }
        } else {
            results, err = inst.InstallFromLocal(fromSource, skillsDest)
        }
    } else {
        results, err = inst.InstallSkills(skillsDest, tags, languages)
    }

    if err != nil {
        return err
    }
    for _, r := range results {
        fmt.Println(r)
    }

    // Install agents
    if !skipAgents && agentsDest != "" {
        fmt.Println("\nInstalling agents...")

        // Copilot requires *.agent.md naming convention
        var nameFunc installer.AgentNameFunc
        if target.Name == "GitHub Copilot" {
            nameFunc = installer.CopilotAgentName
        }

        agentResults, err := inst.InstallAgents(agentsDest, nameFunc)
        if err != nil {
            return err
        }
        for _, r := range agentResults {
            fmt.Println(r)
        }
    }

    // Install commands (if target supports it and project-scoped)
    if !skipCommands && commandsDest != "" && scope == "project" {
        fmt.Println("\nInstalling commands...")
        cmdResults, err := inst.InstallCommands(commandsDest)
        if err != nil {
            return err
        }
        for _, r := range cmdResults {
            fmt.Println(r)
        }
    }

    // Generate config file
    if updateConfig {
        err = generateConfigFile(inst, target)
        if err != nil {
            fmt.Printf("Warning: Could not generate %s: %v\n", target.ConfigPath, err)
        }
    }

    if dryRun {
        fmt.Println("\n(dry run - no files were modified)")
    } else {
        fmt.Println("\nDone! Skills and agents installed successfully.")
    }

    return nil
}
```

**Step 2: Write `generateConfigFile` function**

```go
func generateConfigFile(inst *installer.Installer, target Target) error {
    configPath := filepath.Join(".", target.ConfigPath)

    if dryRun {
        if fileExists(configPath) {
            fmt.Printf("WOULD UPDATE: %s\n", configPath)
        } else {
            fmt.Printf("WOULD CREATE: %s\n", configPath)
        }
        return nil
    }

    dir := filepath.Dir(configPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    // Read the base template from embedded FS
    baseContent, err := fs.ReadFile(content, "templates/CLAUDE-BASE.md")
    if err != nil {
        return fmt.Errorf("reading template: %w", err)
    }

    // For Claude Code, install the full CLAUDE-BASE.md template
    // For other frameworks, generate a simplified config
    var configContent []byte
    switch target.Name {
    case "Claude Code":
        configContent = baseContent
    default:
        // For non-Claude targets, generate a config that:
        // 1. References the installed skills
        // 2. Includes the workflow guidelines
        // 3. Uses framework-appropriate language
        configContent = generateFrameworkConfig(target, baseContent)
    }

    if fileExists(configPath) {
        // Don't overwrite existing config unless --force
        if !force {
            fmt.Printf("SKIP: %s (already exists, use --force to overwrite)\n", configPath)
            return nil
        }
    }

    if err := os.WriteFile(configPath, configContent, 0644); err != nil {
        return err
    }
    fmt.Printf("CREATED: %s\n", configPath)
    return nil
}

func generateFrameworkConfig(target Target, baseContent []byte) []byte {
    // Strip Claude-specific placeholders and adapt for the target framework
    config := string(baseContent)

    // Replace CLAUDE.md-specific header
    header := fmt.Sprintf("# %s - AI Agent Configuration\n\n", target.Name)
    header += fmt.Sprintf("Skills are installed in `%s/`\n", target.SkillsPath)
    if target.AgentsPath != "" {
        header += fmt.Sprintf("Agents are installed in `%s/`\n", target.AgentsPath)
    }

    // Keep the workflow sections (TDD, debugging, pre-push) as they're framework-agnostic
    // but remove Claude-specific template variables
    config = strings.ReplaceAll(config, "{{PROJECT_NAME}}", "Project")
    config = strings.ReplaceAll(config, "{{PROJECT_DESCRIPTION}}", "")
    config = strings.ReplaceAll(config, "{{KEY_DIRECTORIES}}", "")
    config = strings.ReplaceAll(config, "{{TEST_COMMAND}}", "npm test")
    config = strings.ReplaceAll(config, "{{TYPECHECK_COMMAND}}", "npm run typecheck")
    config = strings.ReplaceAll(config, "{{BUILD_COMMAND}}", "npm run build")

    return []byte(header + "\n---\n\n" + config)
}
```

**Step 3: Update `applyConfig` - remove `AgentsPath` reference, add `SkipAgents`**

```go
func applyConfig(cfg *config.Config) {
    if cfg == nil {
        return
    }
    if targetType == "" && cfg.Target != "" {
        targetType = cfg.Target
    }
    if len(tags) == 0 && len(cfg.Tags) > 0 {
        tags = cfg.Tags
    }
    if len(languages) == 0 && len(cfg.Languages) > 0 {
        languages = cfg.Languages
    }
    if !skipClaude && cfg.SkipClaudeMD {
        skipClaude = true
    }
    if fromSource == "" && cfg.From != "" {
        fromSource = cfg.From
    }
}
```

**Step 4: Update `runList` to not reference packs**

Remove pack grouping from `runList`. List skills in alphabetical order with tags shown:

```go
func runList(cmd *cobra.Command, args []string) error {
    inst := installer.New(content, installer.Options{})

    skills, err := inst.ListAllSkills()
    if err != nil {
        return err
    }

    // Apply filters
    var filtered []installer.Skill
    for _, s := range skills {
        if len(tags) > 0 {
            tagMatch := false
            for _, t := range tags {
                for _, st := range s.Tags {
                    if strings.EqualFold(st, t) {
                        tagMatch = true
                    }
                }
            }
            if !tagMatch {
                continue
            }
        }
        filtered = append(filtered, s)
    }

    if len(filtered) == 0 {
        fmt.Println("No skills match the specified filters.")
        return nil
    }

    fmt.Printf("Available skills (%d):\n\n", len(filtered))
    for _, s := range filtered {
        tagsStr := ""
        if len(s.Tags) > 0 {
            tagsStr = " [" + strings.Join(s.Tags, ", ") + "]"
        }
        fmt.Printf("  %-35s %s%s\n", s.Name, truncate(s.Description, 45), tagsStr)
    }

    return nil
}
```

**Step 5: Commit**

```bash
git add main.go
git commit -m "refactor: rewrite runInstall with scope prompt, config generation, command installation"
```

---

## Task 6: Update config.go to match new flag structure

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Update Config struct**

Remove `Packs` and `AgentsPath`. The config is simplified:

```go
type Config struct {
    Target       string   `yaml:"target"`
    Tags         []string `yaml:"tags"`
    Languages    []string `yaml:"languages"`
    SkipClaudeMD bool     `yaml:"skip_claude_md"`
    From         string   `yaml:"from"`
}
```

**Step 2: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor: simplify config struct to match new flag structure"
```

---

## Task 7: Rewrite tests

**Files:**
- Rewrite: `internal/installer/installer_test.go`

**Step 1: Update imports**

```go
import (
    "testing"
    "testing/fstest"
    "path/filepath"
    "os"
)
```

**Step 2: TestParseSkill - plugin-style frontmatter**

```go
func TestParseSkill_FullFrontmatter(t *testing.T) {
    content := []byte("---\nname: test-skill\ndescription: A test skill\nmodel: opus\ntags: [testing, quality]\nlanguages: [go, any]\n---\n# Test Skill")
    skill, err := parseSkill(content)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if skill.Name != "test-skill" { t.Errorf("got name %q", skill.Name) }
    if skill.Model != "opus" { t.Errorf("got model %q", skill.Model) }
    if len(skill.Tags) != 2 { t.Errorf("got %d tags", len(skill.Tags)) }
}

func TestParseSkill_MinimalFrontmatter(t *testing.T) {
    // Plugin skills often only have name and description
    content := []byte("---\nname: my-skill\ndescription: Does things\n---\n# Content")
    skill, err := parseSkill(content)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if skill.Name != "my-skill" { t.Errorf("got name %q", skill.Name) }
    if skill.Model != "" { t.Errorf("got model %q, want empty", skill.Model) }
}

func TestParseSkill_ExtraFields(t *testing.T) {
    // Plugin skills may have allowed-tools, argument-hint - should be ignored
    content := []byte("---\nname: browser\ndescription: Browse\nallowed-tools: [Bash]\nargument-hint: URL\n---\n# Browser")
    skill, err := parseSkill(content)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if skill.Name != "browser" { t.Errorf("got name %q", skill.Name) }
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
```

**Step 3: TestDiscoverSkills**

```go
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
}
```

**Step 4: TestInstallSkills - copies entire directories**

```go
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
```

**Step 5: TestInstallAgents (with and without naming function)**

```go
func TestInstallAgents(t *testing.T) {
    testFS := fstest.MapFS{
        "agents/debugger.md": &fstest.MapFile{
            Data: []byte("---\nname: debugger\ndescription: Debug\n---\n# Debugger"),
        },
        "agents/reviewer.md": &fstest.MapFile{
            Data: []byte("---\nname: reviewer\ndescription: Review\n---\n# Reviewer"),
        },
    }

    t.Run("default naming", func(t *testing.T) {
        tmpDir := t.TempDir()
        inst := New(testFS, Options{})
        results, err := inst.InstallAgents(tmpDir, nil)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(results) != 2 {
            t.Fatalf("expected 2 results, got %d", len(results))
        }
        // Verify original names kept
        if _, err := os.Stat(filepath.Join(tmpDir, "debugger.md")); os.IsNotExist(err) {
            t.Error("debugger.md not created")
        }
    })

    t.Run("copilot naming", func(t *testing.T) {
        tmpDir := t.TempDir()
        inst := New(testFS, Options{})
        results, err := inst.InstallAgents(tmpDir, CopilotAgentName)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(results) != 2 {
            t.Fatalf("expected 2 results, got %d", len(results))
        }
        // Verify *.agent.md naming
        if _, err := os.Stat(filepath.Join(tmpDir, "debugger.agent.md")); os.IsNotExist(err) {
            t.Error("debugger.agent.md not created")
        }
    })
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
```
```

**Step 6: TestInstallCommands**

```go
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
```

**Step 7: TestWriteFile scenarios** (keep existing tests, update paths)

**Step 8: TestFilterByTags**

```go
func TestFilterByTags(t *testing.T) {
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

    // Filter by tag
    results, err := inst.InstallSkills(tmpDir, []string{"workflow"}, nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Only "tagged" should be installed
    if len(results) != 1 {
        t.Fatalf("expected 1 result, got %d: %v", len(results), results)
    }
}
```

**Step 9: Run tests**

```bash
go test ./... -v
```

Expected: All tests pass.

**Step 10: Commit**

```bash
git add internal/installer/installer_test.go
git commit -m "test: rewrite tests for directory-based skills and fs.FS interface"
```

---

## Task 8: Add tags to existing plugin skills

For tag-based filtering to work, existing plugin skills need `tags` in their SKILL.md frontmatter. Add tags to each skill based on its category.

**Files:**
- Modify: All 32 `skills/*/SKILL.md` files (add `tags:` line to frontmatter)

**Tag taxonomy:**

| Category | Tag | Skills |
|----------|-----|--------|
| Core workflow | `workflow` | systematic-debugging, writing-plans, executing-plans, brainstorming, using-superpowers |
| Code quality | `quality` | code-simplifier, requesting-code-review, receiving-code-review, error-handling-patterns |
| Development | `development` | dispatching-parallel-agents, subagent-driven-development, using-git-worktrees, finishing-a-development-branch |
| Testing | `testing` | javascript-testing-patterns |
| Framework | `framework` | adonisjs-best-practices, better-auth-best-practices, sqlite-database-expert, turso-best-practices, create-auth-skill |
| Design | `design` | frontend-design, ui-design, design-principles |
| Search | `search` | code-search |
| Authoring | `authoring` | skill-creator, writing-skills |
| Marketing | `marketing` | copywriting, marketing-psychology, programmatic-seo |
| Other | `tools` | agent-browser, baoyu-article-illustrator |
| Architecture | `architecture` | api-design-principles, architecture-decision-records |

**Step 1: For each skill, add a `tags:` line after the `description:` line in the YAML frontmatter**

Example for `skills/systematic-debugging/SKILL.md`:
```yaml
---
name: systematic-debugging
description: Use when encountering any bug...
tags: [workflow, debugging]
---
```

**Step 2: Verify all skills parse correctly**

```bash
go build -o skill-installer . && ./skill-installer list
```

Expected: All 32 skills listed with tags.

**Step 3: Commit**

```bash
git add skills/
git commit -m "feat: add tags to all skill frontmatter for filtering"
```

---

## Task 9: Update README.md

**Files:**
- Modify: `README.md`

**Step 1: Rewrite README**

Structure:
1. **Overview** - What this is (skills + agents + commands + CLI for multiple frameworks)
2. **Installation**
   - **As Claude Code Plugin** (symlink / `--plugin-dir` — existing instructions)
   - **Via CLI** (download binary or `go install`, run `skill-installer`)
3. **CLI Usage**
   - Install skills (interactive, non-interactive, with filters, from custom source)
   - List skills
   - Create new skill
   - Scope (project vs global)
4. **Supported Frameworks** - Table with all 5 targets
5. **Contents** - Skills table, agents table, commands, templates (existing content)
6. **Configuration** - `.skill-installer.yaml`
7. **Building from Source** - `make build`, `make test`
8. **Attribution & Contributing**

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README for merged CLI + plugin repo"
```

---

## Task 10: Update plugin.json

**Files:**
- Modify: `.claude-plugin/plugin.json`

**Step 1: Bump version and update description**

```json
{
  "name": "futuregerald-claude-plugin",
  "description": "Portable skills, agents, and commands for Claude Code and other AI IDEs - includes a CLI installer for Cursor, Copilot, OpenCode, and VS Code",
  "version": "2.0.0",
  "author": {
    "name": "futuregerald"
  },
  "license": "MIT"
}
```

**Step 2: Commit**

```bash
git add .claude-plugin/plugin.json
git commit -m "chore: bump to v2.0.0 with CLI integration"
```

---

## Task 11: Build and smoke-test

**Step 1: Build the binary**

```bash
make build
```

Expected: `skill-installer` binary created.

**Step 2: List skills**

```bash
./skill-installer list
```

Expected: Lists all 32 skills with names, descriptions, and tags.

**Step 3: List skills filtered by tag**

```bash
./skill-installer list --tag workflow
```

Expected: Only workflow-tagged skills shown.

**Step 4: Dry-run install for each target**

```bash
./skill-installer --target claude --dry-run --yes
./skill-installer --target copilot --dry-run --yes
./skill-installer --target cursor --dry-run --yes
./skill-installer --target opencode --dry-run --yes
./skill-installer --target vscode --dry-run --yes
```

Expected: Each shows WOULD CREATE lines for skills, agents, and (where applicable) commands in the target's directories.

**Step 5: Test actual installation to a temp directory**

```bash
mkdir /tmp/test-install && cd /tmp/test-install
/path/to/skill-installer --target claude --yes
ls -la .claude/skills/
ls -la .claude/agents/
ls -la .claude/commands/
cat CLAUDE.md
cd - && rm -rf /tmp/test-install
```

Expected: Skills installed as directories with SKILL.md, agents as .md files, commands as directories, CLAUDE.md generated.

**Step 6: Verify each skill directory was copied completely**

```bash
# Pick a skill with references
ls .claude/skills/systematic-debugging/
```

Expected: SKILL.md, defense-in-depth.md, root-cause-tracing.md, etc.

**Step 7: Verify Claude Code plugin still works**

```bash
claude --plugin-dir /Users/geraldonyango/Documents/dev/futuregerald-claude-plugin
```

Expected: Skills load and are usable via `/futuregerald-claude-plugin:systematic-debugging`.

**Step 8: Run full test suite**

```bash
make test
```

Expected: All tests pass.

**Step 9: Create new skill with `init`**

```bash
./skill-installer init test-skill --desc "A test skill" --tag custom
ls test-skill/
cat test-skill/SKILL.md
rm -rf test-skill/
```

Expected: Creates `test-skill/SKILL.md` with proper frontmatter.

**Step 10: Commit any fixes found during smoke testing**

---

## Task 12: Clean up

**Step 1: Remove `skill-zips/` directory**

```bash
rm -rf skill-zips/
```

**Step 2: Verify no stale files**

```bash
git status
```

**Step 3: Final commit**

```bash
git add -A
git commit -m "chore: clean up merged repo, remove skill-zips"
```

---

## Key Decisions

1. **Plugin content is source of truth** - No skills, agents, templates, or content brought from the installer.
2. **Packs replaced by tags** - More flexible, doesn't require directory restructuring.
3. **`fs.FS` interface** - Makes installer testable with `fstest.MapFS`.
4. **`path.Join` for embedded FS, `filepath.Join` for OS paths** - Prevents Windows breakage.
5. **Scope prompt (project vs global)** - Copilot and Claude Code support both; CLI asks users.
6. **Commands installed by CLI** - Claude Code commands get installed to target's command path where supported.
7. **Zip-slip fix** - Path traversal vulnerability fixed in `extractTarGz`.
8. **`runInit` creates directories** - `skill-installer init` now creates `name/SKILL.md` not flat files.
9. **Agent installation** - Agent .md files installed to target's agent directory. For Copilot, agents go to `.github/` as `*.agent.md` files per Copilot convention.
10. **Framework config generation** - CLAUDE.md for Claude, .cursorrules for Cursor, copilot-instructions.md for Copilot. Template-based with framework-appropriate adaptations.

---

## Staff Engineer Review: Issues Addressed

### Round 1 Issues (all resolved)

| # | Issue | Resolution |
|---|-------|------------|
| 1 | `filepath.Join` for embed paths | All embedded FS paths use `path.Join`; OS paths use `filepath.Join` |
| 2 | `code-simplifier` missing SKILL.md | Added to Task 2 rename list |
| 3 | Contradictory "remove InstallFromLocal" | Kept and adapted all three methods (Task 4 Step 11) |
| 4 | `strings.Title` deprecated | Removed `runPacks`; added `titleCase` helper in both packages |
| 5 | `embed.FS` → `fs.FS` timing | Done in Task 4 Step 2 (where installer.go is rewritten) |
| 6 | `fs.ReadFile` free function consistency | All methods use `fs.ReadFile(i.fsys, ...)` pattern |
| 7 | `runInit` creates flat files | Updated in Task 3 Step 13 to create `name/SKILL.md` |
| 8 | No `.gitignore` | Created in Task 1 Step 6 |
| 9 | `generateGenericConfig` undefined | Defined as `generateFrameworkConfig` in Task 5 Step 2 |
| 10 | SKILL.md count wrong | Fixed by adding code-simplifier to rename list |
| 11 | `Makefile` uses `mv` | Changed to `install -m 755` |
| 12 | Config struct `AgentsPath` | Removed in Task 6 |
| 13 | Missing `io/fs` and `path` imports | Explicitly added in Task 4 Step 1 |
| 14 | `ListPacks()` broken | Removed entirely (packs dropped) |
| 15 | No CI step | Deferred to follow-up (user can decide on GitHub Actions) |
| 16 | Zip-slip vulnerability | Fixed in Task 4 Step 12 |

### Round 2 Issues (all resolved)

| # | Issue | Resolution |
|---|-------|------------|
| R2-1 | `strings.Title` in `GenerateSkillTemplate` | Added `titleCase` helper to installer package (Task 4 Step 15a) |
| R2-2 | Copilot `*.agent.md` renaming not implemented | `InstallAgents` now accepts `AgentNameFunc`; `CopilotAgentName` converts `debugger.md` → `debugger.agent.md` (Task 4 Step 7); caller passes it for Copilot target (Task 5 Step 1) |
| R2-3 | `--global` flag declared but not wired | Added `globalInstall` variable, wired to `askScope` (Task 3 Steps 5, 8) |
| R2-4 | `skipAgents`/`skipCommands` variables not declared | Added to `var` block with flag bindings (Task 3 Step 5) |
