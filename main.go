package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/config"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/installer"
	"github.com/spf13/cobra"
)

//go:embed skills/*.md skills/**/*.md templates/*.md agents/*.md
var content embed.FS

var (
	version     = "0.2.0"
	force       bool
	dryRun      bool
	nonInteract bool
	targetType  string
	agentsPath  string
	skipClaude  bool
	packs       []string
	tags        []string
	languages   []string
	fromSource  string
	configFile  string
	showAll     bool
)

// Target represents an installation target (IDE/tool).
type Target struct {
	Name       string
	SkillsPath string
	ConfigPath string
}

var targets = map[string]Target{
	"claude": {
		Name:       "Claude Code",
		SkillsPath: ".claude/skills",
		ConfigPath: "CLAUDE.md",
	},
	"opencode": {
		Name:       "OpenCode",
		SkillsPath: ".opencode/skills",
		ConfigPath: "",
	},
	"cursor": {
		Name:       "Cursor",
		SkillsPath: ".cursor/skills",
		ConfigPath: ".cursorrules",
	},
	"vscode": {
		Name:       "VS Code (with Claude extension)",
		SkillsPath: ".vscode/claude/skills",
		ConfigPath: "",
	},
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "skill-installer",
		Short: "Install AI coding assistant skills",
		Long: `A CLI tool to install AI coding assistant skills for various IDEs and tools.

Supported targets:
  - Claude Code (.claude/skills)
  - OpenCode (.opencode/skills)
  - Cursor (.cursor/skills)
  - VS Code with Claude extension (.vscode/claude/skills)

Available packs: core, go, python, typescript, rust

Configuration can be stored in .skill-installer.yaml`,
		RunE: runInstall,
	}

	// Install flags
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without making changes")
	rootCmd.Flags().BoolVarP(&nonInteract, "yes", "y", false, "Non-interactive mode with defaults")
	rootCmd.Flags().StringVarP(&targetType, "target", "t", "", "Target: claude, opencode, cursor, vscode")
	rootCmd.Flags().StringVarP(&agentsPath, "agents", "a", "", "Path for agents.md")
	rootCmd.Flags().BoolVar(&skipClaude, "skip-claude-md", false, "Skip updating CLAUDE.md")
	rootCmd.Flags().StringSliceVarP(&packs, "pack", "p", nil, "Packs to install (core, go, python, typescript, rust)")
	rootCmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter skills by tags")
	rootCmd.Flags().StringSliceVar(&languages, "lang", nil, "Filter skills by language")
	rootCmd.Flags().StringVar(&fromSource, "from", "", "Install from source (local path, git URL, or URL)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("skill-installer v%s\n", version)
		},
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Long: `List available skills with optional filtering.

Examples:
  skill-installer list                    # List core skills
  skill-installer list --all              # List all skills including packs
  skill-installer list --pack go          # List Go pack skills
  skill-installer list --tag testing      # List skills tagged with 'testing'
  skill-installer list --lang python      # List Python-compatible skills`,
		RunE: runList,
	}
	listCmd.Flags().BoolVar(&showAll, "all", false, "Show all skills including language packs")
	listCmd.Flags().StringSliceVarP(&packs, "pack", "p", nil, "Filter by pack")
	listCmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tags")
	listCmd.Flags().StringSliceVar(&languages, "lang", nil, "Filter by language")

	// Packs command
	packsCmd := &cobra.Command{
		Use:   "packs",
		Short: "List available language packs",
		RunE:  runPacks,
	}

	// Init command
	initCmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Create a new skill from template",
		Long: `Create a new skill file with proper frontmatter.

Examples:
  skill-installer init my-skill
  skill-installer init my-skill --model opus --tag quality,review`,
		Args: cobra.ExactArgs(1),
		RunE: runInit,
	}
	var initModel string
	var initTags []string
	var initLangs []string
	var initDesc string
	initCmd.Flags().StringVar(&initModel, "model", "sonnet", "Model to use (haiku, sonnet, opus)")
	initCmd.Flags().StringSliceVar(&initTags, "tag", nil, "Tags for the skill")
	initCmd.Flags().StringSliceVar(&initLangs, "lang", []string{"any"}, "Languages for the skill")
	initCmd.Flags().StringVarP(&initDesc, "desc", "d", "", "Description of the skill")

	rootCmd.AddCommand(versionCmd, listCmd, packsCmd, initCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runInstall(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Load config file if present
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	applyConfig(cfg)

	// Get target
	target, err := getTarget(reader)
	if err != nil {
		return err
	}

	// Get agents.md location
	agentsLocation, err := getAgentsLocation(reader)
	if err != nil {
		return err
	}

	// For Claude Code, ask about CLAUDE.md
	updateClaudeMD := false
	if target.Name == "Claude Code" && !skipClaude {
		updateClaudeMD, err = askUpdateClaudeMD(reader)
		if err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println("Installing skills...")

	inst := installer.New(content, installer.Options{
		Force:  force,
		DryRun: dryRun,
	})

	var results []string

	// Install from custom source
	if fromSource != "" {
		destDir := filepath.Join(".", target.SkillsPath)
		if strings.HasPrefix(fromSource, "http://") || strings.HasPrefix(fromSource, "https://") {
			if strings.Contains(fromSource, "github.com") || strings.Contains(fromSource, "gitlab.com") {
				results, err = inst.InstallFromGit(fromSource, destDir)
			} else {
				results, err = inst.InstallFromURL(fromSource, destDir)
			}
		} else {
			results, err = inst.InstallFromLocal(fromSource, destDir)
		}
	} else if len(tags) > 0 || len(languages) > 0 {
		// Install filtered by tags/languages
		results, err = inst.InstallSkillsFiltered(".", target.SkillsPath, tags, languages)
	} else {
		// Install by packs
		results, err = inst.InstallSkills(".", target.SkillsPath, packs)
	}

	if err != nil {
		return err
	}
	for _, r := range results {
		fmt.Println(r)
	}

	// Install agents.md
	result, err := inst.InstallAgentsMD(agentsLocation)
	if err != nil {
		return err
	}
	fmt.Println(result)

	// Update CLAUDE.md if requested
	if updateClaudeMD && !dryRun {
		err = updateClaudeMDFile(agentsLocation)
		if err != nil {
			fmt.Printf("Warning: Could not update CLAUDE.md: %v\n", err)
		} else {
			fmt.Println("UPDATED: CLAUDE.md (added agents.md reference)")
		}
	} else if updateClaudeMD && dryRun {
		fmt.Println("WOULD UPDATE: CLAUDE.md (add agents.md reference)")
	}

	if dryRun {
		fmt.Println("\n(dry run - no files were modified)")
	} else {
		fmt.Println("\nDone! Skills installed successfully.")
	}

	return nil
}

func loadConfig() (*config.Config, error) {
	if configFile != "" {
		return config.LoadFile(configFile)
	}
	return config.Load(".")
}

func applyConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if targetType == "" && cfg.Target != "" {
		targetType = cfg.Target
	}
	if len(packs) == 0 && len(cfg.Packs) > 0 {
		packs = cfg.Packs
	}
	if len(tags) == 0 && len(cfg.Tags) > 0 {
		tags = cfg.Tags
	}
	if len(languages) == 0 && len(cfg.Languages) > 0 {
		languages = cfg.Languages
	}
	if agentsPath == "" && cfg.AgentsPath != "" {
		agentsPath = cfg.AgentsPath
	}
	if !skipClaude && cfg.SkipClaudeMD {
		skipClaude = true
	}
	if fromSource == "" && cfg.From != "" {
		fromSource = cfg.From
	}
}

func getTarget(reader *bufio.Reader) (Target, error) {
	if targetType != "" {
		if t, ok := targets[targetType]; ok {
			return t, nil
		}
		return Target{}, fmt.Errorf("unknown target: %s", targetType)
	}

	if nonInteract {
		return targets["claude"], nil
	}

	fmt.Println("Where would you like to install the skills?")
	fmt.Println()
	options := []string{"claude", "opencode", "cursor", "vscode"}
	for i, key := range options {
		t := targets[key]
		fmt.Printf("  %d) %s (%s)\n", i+1, t.Name, t.SkillsPath)
	}
	fmt.Println()
	fmt.Print("Enter choice [1]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return Target{}, err
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return targets["claude"], nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		return Target{}, fmt.Errorf("invalid choice: %s", input)
	}

	return targets[options[choice-1]], nil
}

func getAgentsLocation(reader *bufio.Reader) (string, error) {
	if agentsPath != "" {
		return agentsPath, nil
	}

	if nonInteract {
		return ".", nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	fmt.Printf("\nWhere should agents.md be created?\n")
	fmt.Printf("Enter path [%s]: ", cwd)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return ".", nil
	}

	return input, nil
}

func askUpdateClaudeMD(reader *bufio.Reader) (bool, error) {
	if nonInteract {
		return true, nil
	}

	fmt.Print("\nUpdate CLAUDE.md to reference agents.md? [Y/n]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "" || input == "y" || input == "yes", nil
}

func updateClaudeMDFile(agentsDir string) error {
	claudePath := filepath.Join(agentsDir, "CLAUDE.md")
	agentsRef := "\n\nSee [agents.md](agents.md) for AI agent guidelines and available skills.\n"

	content, err := os.ReadFile(claudePath)
	if err != nil {
		if os.IsNotExist(err) {
			newContent := "# Project Guidelines\n" + agentsRef
			return os.WriteFile(claudePath, []byte(newContent), 0644)
		}
		return err
	}

	if strings.Contains(string(content), "agents.md") {
		return nil
	}

	return os.WriteFile(claudePath, append(content, []byte(agentsRef)...), 0644)
}

func runList(cmd *cobra.Command, args []string) error {
	inst := installer.New(content, installer.Options{})

	var skills []installer.Skill
	var err error

	if showAll || len(packs) > 0 || len(tags) > 0 || len(languages) > 0 {
		skills, err = inst.ListAllSkills()
	} else {
		skills, err = inst.ListSkills()
	}
	if err != nil {
		return err
	}

	// Filter skills
	var filtered []installer.Skill
	for _, s := range skills {
		if len(packs) > 0 {
			packMatch := false
			for _, p := range packs {
				if (p == "core" && s.Pack == "") || s.Pack == p {
					packMatch = true
					break
				}
			}
			if !packMatch {
				continue
			}
		}

		if len(tags) > 0 {
			tagMatch := false
			for _, t := range tags {
				for _, st := range s.Tags {
					if strings.EqualFold(st, t) {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue
			}
		}

		if len(languages) > 0 {
			langMatch := false
			for _, l := range languages {
				for _, sl := range s.Languages {
					if strings.EqualFold(sl, l) || strings.EqualFold(sl, "any") {
						langMatch = true
						break
					}
				}
				if langMatch {
					break
				}
			}
			if !langMatch {
				continue
			}
		}

		filtered = append(filtered, s)
	}

	if len(filtered) == 0 {
		fmt.Println("No skills match the specified filters.")
		return nil
	}

	fmt.Println("Available skills:")
	fmt.Println()

	// Group by pack
	packSkills := make(map[string][]installer.Skill)
	for _, s := range filtered {
		pack := s.Pack
		if pack == "" {
			pack = "core"
		}
		packSkills[pack] = append(packSkills[pack], s)
	}

	packOrder := []string{"core", "go", "python", "typescript", "rust"}
	for _, pack := range packOrder {
		skills, ok := packSkills[pack]
		if !ok || len(skills) == 0 {
			continue
		}

		fmt.Printf("[%s]\n", pack)
		for _, s := range skills {
			tagsStr := ""
			if len(s.Tags) > 0 {
				tagsStr = " [" + strings.Join(s.Tags[:min(3, len(s.Tags))], ", ")
				if len(s.Tags) > 3 {
					tagsStr += ", ..."
				}
				tagsStr += "]"
			}
			fmt.Printf("  %-20s %-8s %s%s\n", s.Name, "("+s.Model+")", truncate(s.Description, 40), tagsStr)
		}
		fmt.Println()
	}

	return nil
}

func runPacks(cmd *cobra.Command, args []string) error {
	inst := installer.New(content, installer.Options{})

	packList, err := inst.ListPacks()
	if err != nil {
		return err
	}

	fmt.Println("Available packs:")
	fmt.Println()
	fmt.Println("  core         Core skills (always included by default)")
	for _, p := range packList {
		fmt.Printf("  %-12s Language-specific skills for %s\n", p, strings.Title(p))
	}
	fmt.Println()
	fmt.Println("Install specific packs:")
	fmt.Println("  skill-installer --pack core,go,python")

	return nil
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]
	model, _ := cmd.Flags().GetString("model")
	initTags, _ := cmd.Flags().GetStringSlice("tag")
	initLangs, _ := cmd.Flags().GetStringSlice("lang")
	desc, _ := cmd.Flags().GetString("desc")

	if desc == "" {
		desc = fmt.Sprintf("Custom skill for %s", name)
	}
	if len(initTags) == 0 {
		initTags = []string{"custom"}
	}

	content := installer.GenerateSkillTemplate(name, desc, model, initTags, initLangs)

	filename := name + ".md"
	if !strings.HasSuffix(filename, ".md") {
		filename = name + ".md"
	}

	if fileExists(filename) && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", filename)
	}

	if dryRun {
		fmt.Printf("WOULD CREATE: %s\n", filename)
		fmt.Println("\nContent preview:")
		fmt.Println(content)
		return nil
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	fmt.Printf("CREATED: %s\n", filename)
	fmt.Println("\nEdit the file to customize your skill, then move it to your skills directory.")

	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
