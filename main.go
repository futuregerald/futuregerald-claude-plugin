package main

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/config"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/installer"
	"github.com/spf13/cobra"
)

const (
	modeFullInstall = "full"
	modeConfigOnly  = "config-only"
	modeAgentsOnly  = "agents-only"
)

//go:embed all:skills all:agents all:templates all:commands
var content embed.FS

var (
	version       = "3.0.0"
	force         bool
	dryRun        bool
	nonInteract   bool
	targetType    string
	skipClaude    bool
	tags          []string
	languages     []string
	fromSource    string
	configFile    string
	skipAgents    bool
	skipCommands  bool
	globalInstall bool
	installMode   string
)

// Target represents an installation target (IDE/tool).
type Target struct {
	Name             string
	SkillsPath       string
	AgentsPath       string
	CommandsPath     string
	ConfigPath       string
	GlobalSkillsPath string
	GlobalAgentsPath string
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

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
		AgentsPath:       ".github",
		CommandsPath:     "",
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

func main() {
	rootCmd := &cobra.Command{
		Use:   "skill-installer",
		Short: "Install AI coding assistant skills",
		Long: `A CLI tool to install AI coding assistant skills for various IDEs and tools.

Supported targets:
  - Claude Code (.claude/skills)
  - GitHub Copilot (.github/skills)
  - OpenCode (.opencode/skills)
  - Cursor (.cursor/skills)
  - VS Code with Claude extension (.vscode/claude/skills)

Configuration can be stored in .skill-installer.yaml`,
		RunE: runInstall,
	}

	// Install flags
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without making changes")
	rootCmd.Flags().BoolVarP(&nonInteract, "yes", "y", false, "Non-interactive mode with defaults")
	rootCmd.Flags().StringVarP(&targetType, "target", "t", "", "Target: claude, copilot, opencode, cursor, vscode")
	rootCmd.Flags().BoolVar(&skipClaude, "skip-claude-md", false, "Skip updating CLAUDE.md")
	rootCmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter skills by tags")
	rootCmd.Flags().StringSliceVar(&languages, "lang", nil, "Filter skills by language")
	rootCmd.Flags().StringVar(&fromSource, "from", "", "Install from source (local path, git URL, or URL)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.Flags().BoolVar(&skipAgents, "skip-agents", false, "Skip installing agents")
	rootCmd.Flags().BoolVar(&skipCommands, "skip-commands", false, "Skip installing commands")
	rootCmd.Flags().BoolVar(&globalInstall, "global", false, "Install to global/user-level directory")
	rootCmd.Flags().StringVarP(&installMode, "mode", "m", "", "Installation mode: full, config-only, agents-only")

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
  skill-installer list                    # List all skills
  skill-installer list --tag testing      # List skills tagged with 'testing'
  skill-installer list --lang python      # List Python-compatible skills`,
		RunE: runList,
	}
	listCmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tags")
	listCmd.Flags().StringSliceVar(&languages, "lang", nil, "Filter by language")

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

	rootCmd.AddCommand(versionCmd, listCmd, initCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

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

func generateConfigFile(_ *installer.Installer, target Target) error {
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
		configContent = generateFrameworkConfig(target, baseContent)
	}

	if fileExists(configPath) {
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
	header := fmt.Sprintf("# %s - AI Agent Configuration\n\n", target.Name)
	header += fmt.Sprintf("Skills are installed in `%s/`\n", target.SkillsPath)
	if target.AgentsPath != "" {
		header += fmt.Sprintf("Agents are installed in `%s/`\n", target.AgentsPath)
	}

	config := string(baseContent)
	config = strings.ReplaceAll(config, "{{PROJECT_NAME}}", "Project")
	config = strings.ReplaceAll(config, "{{PROJECT_DESCRIPTION}}", "")
	config = strings.ReplaceAll(config, "{{KEY_DIRECTORIES}}", "")
	config = strings.ReplaceAll(config, "{{TEST_COMMAND}}", "npm test")
	config = strings.ReplaceAll(config, "{{TYPECHECK_COMMAND}}", "npm run typecheck")
	config = strings.ReplaceAll(config, "{{BUILD_COMMAND}}", "npm run build")

	return []byte(header + "\n---\n\n" + config)
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
	if installMode == "" && cfg.Mode != "" {
		installMode = cfg.Mode
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
	options := []string{"claude", "copilot", "cursor", "opencode", "vscode"}
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

func askScope(reader *bufio.Reader, target Target) (string, error) {
	if target.GlobalSkillsPath == "" {
		return "project", nil
	}
	if globalInstall {
		return "global", nil
	}
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

	skillContent := installer.GenerateSkillTemplate(name, desc, model, initTags, initLangs)

	skillDir := name
	filename := filepath.Join(skillDir, "SKILL.md")

	if fileExists(filename) && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", filename)
	}

	if dryRun {
		fmt.Printf("WOULD CREATE: %s\n", filename)
		return nil
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", skillDir, err)
	}

	if err := os.WriteFile(filename, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	fmt.Printf("CREATED: %s\n", filename)
	fmt.Println("\nEdit the file to customize your skill, then move the directory to your skills location.")
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

