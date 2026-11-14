package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/backup"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/config"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/installer"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/state"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/tui"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/updater"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	modeFullInstall = "full"
	modeConfigOnly  = "config-only"
	modeAgentsOnly  = "agents-only"
)

const (
	cbMemVersion = "v0.4.10"
	cbMemRepo    = "DeusData/codebase-memory-mcp"
	cbMemBinName = "codebase-memory-mcp"
)

var content fs.FS = catalog.FS

var (
	version        = "4.0.0"
	force          bool
	dryRun         bool
	nonInteract    bool
	targetType     string
	trackName      string
	selectedSkills []string
	backupFlag     = true
	noBackupFlag   bool
	backupExplicit bool
	skipClaude     bool
	tags           []string
	languages      []string
	fromSource     string
	configFile     string
	skipAgents     bool
	skipCommands   bool
	globalInstall  bool
	installMode    string
	listLong       bool
	updateCheck    bool
	updateApply    bool
	updateSelf     bool
	updateJSON     bool
	restoreID      string
	restoreConfig  bool
	updateRunner   updater.Runner
)

type Target struct {
	Name             string
	Harness          string
	SkillsPath       string
	AgentsPath       string
	CommandsPath     string
	ConfigPath       string
	GlobalSkillsPath string
	GlobalAgentsPath string
}

var targetOrder = []string{
	installer.HarnessClaude,
	installer.HarnessAntigravity,
	installer.HarnessZed,
	installer.HarnessOpenCode,
}

var targets = map[string]Target{
	installer.HarnessClaude: {
		Name:         "Claude Code",
		Harness:      installer.HarnessClaude,
		SkillsPath:   ".claude/skills",
		AgentsPath:   ".claude/agents",
		CommandsPath: ".claude/commands",
		ConfigPath:   "CLAUDE.md",
	},
	installer.HarnessAntigravity: {
		Name:         "Antigravity",
		Harness:      installer.HarnessAntigravity,
		SkillsPath:   ".agents/skills",
		AgentsPath:   ".agents/agents",
		CommandsPath: "",
		ConfigPath:   "GEMINI.md",
	},
	installer.HarnessZed: {
		Name:         "Zed Editor",
		Harness:      installer.HarnessZed,
		SkillsPath:   ".agents/skills",
		AgentsPath:   "",
		CommandsPath: "",
		ConfigPath:   "AGENTS.md",
	},
	installer.HarnessOpenCode: {
		Name:         "OpenCode",
		Harness:      installer.HarnessOpenCode,
		SkillsPath:   ".opencode/skills",
		AgentsPath:   ".opencode/agents",
		CommandsPath: ".opencode/commands",
		ConfigPath:   "AGENTS.md",
	},
}

func resolveBackupChoice(cmd *cobra.Command, reader *bufio.Reader) (bool, error) {
	if noBackupFlag {
		return false, nil
	}
	if cmd != nil {
		if cmd.Flags().Changed("no-backup") && noBackupFlag {
			return false, nil
		}
		if cmd.Flags().Changed("backup") {
			return backupFlag, nil
		}
	}
	if backupExplicit {
		return backupFlag, nil
	}
	if nonInteract || reader == nil {
		return backupFlag, nil
	}
	return askBackupConfirmation(reader)
}

func askBackupConfirmation(reader *bufio.Reader) (bool, error) {
	fmt.Print("\nSave a safety backup before writing files? [Y/n]: ")
	input, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return true, err
	}
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "n" || trimmed == "no" {
		return false, nil
	}
	return true, nil
}

func main() {
	if exe, err := os.Executable(); err == nil {
		platform.SweepStaleExecutable(exe)
	}

	rootCmd := buildRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "skill-installer",
		Short: "Spacious graphical installer & updater for AI coding and product workflows",
		Long: `skill-installer v4.0 — Curated workflows, slash commands, subagents, and skills
for Google Antigravity, Claude Code, Zed Editor, and OpenCode across Windows, Linux, and macOS.

Run without flags in a terminal to launch the guided graphical TUI, or pass flags for CLI/CI runs.`,
		RunE: runInstall,
	}

	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files")
	rootCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without making changes")
	rootCmd.Flags().BoolVarP(&nonInteract, "yes", "y", false, "Non-interactive mode with defaults")
	rootCmd.Flags().StringVarP(&targetType, "target", "t", "", "Target harness: claude, antigravity, zed, opencode")
	rootCmd.Flags().StringVar(&trackName, "track", "", "Curated track: opinionated, basics, product, copy-polish, custom")
	rootCmd.Flags().StringSliceVar(&selectedSkills, "skill", nil, "Specific skill names to install (implies custom selection)")
	rootCmd.Flags().BoolVar(&backupFlag, "backup", true, "Save a safety backup snapshot before overwriting files (default true)")
	rootCmd.Flags().BoolVar(&noBackupFlag, "no-backup", false, "Skip creating a safety backup snapshot")
	rootCmd.MarkFlagsMutuallyExclusive("backup", "no-backup")
	rootCmd.Flags().BoolVar(&skipClaude, "skip-claude-md", false, "Skip updating project/global config file (CLAUDE.md, GEMINI.md, AGENTS.md)")
	rootCmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter skills by tags")
	rootCmd.Flags().StringSliceVar(&languages, "lang", nil, "Filter skills by language")
	rootCmd.Flags().StringVar(&fromSource, "from", "", "Install from external source (local path, git URL, or tarball URL)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.Flags().BoolVar(&skipAgents, "skip-agents", false, "Skip installing subagents")
	rootCmd.Flags().BoolVar(&skipCommands, "skip-commands", false, "Skip installing slash commands")
	rootCmd.Flags().BoolVar(&globalInstall, "global", false, "Install to global user-level directory")
	rootCmd.Flags().StringVarP(&installMode, "mode", "m", "", "Installation mode: full, config-only, agents-only")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("skill-installer v%s\n", version)
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List curated tracks and skills with slash commands and descriptions",
		RunE:  runList,
	}
	listCmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tags")
	listCmd.Flags().StringSliceVar(&languages, "lang", nil, "Filter by language")
	listCmd.Flags().StringVarP(&targetType, "target", "t", "", "Filter skills for target harness (claude, antigravity, zed, opencode)")
	listCmd.Flags().StringVar(&trackName, "track", "", "Filter skills by curated track (opinionated, basics, product, copy-polish)")
	listCmd.Flags().BoolVar(&listLong, "long", false, "Show full long descriptions and starter prompts")

	guideCmd := &cobra.Command{
		Use:   "guide",
		Short: "Interactive plain-English guide to slash commands, skills, and tracks",
		RunE:  runGuide,
	}
	guideCmd.Flags().StringVarP(&targetType, "target", "t", "", "Target harness (claude, antigravity, zed, opencode)")
	guideCmd.Flags().StringVar(&trackName, "track", "opinionated", "Track to preview in guide")
	guideCmd.Flags().BoolVarP(&nonInteract, "yes", "y", false, "Print formatted guide to stdout without launching TUI")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and apply live skill updates from main via gh CLI",
		RunE:  runUpdate,
	}
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "Only check whether updates are available without applying")
	updateCmd.Flags().BoolVar(&updateApply, "apply", false, "Apply available updates from main")
	updateCmd.Flags().BoolVar(&updateSelf, "self", false, "Upgrade the skill-installer binary from GitHub Releases")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Emit machine-readable JSON output for /plugin-update")
	updateCmd.Flags().StringVarP(&targetType, "target", "t", "", "Target harness to update (claude, antigravity, zed, opencode)")
	updateCmd.Flags().BoolVar(&globalInstall, "global", false, "Update global installation")
	updateCmd.Flags().BoolVar(&backupFlag, "backup", true, "Save a safety backup before applying update")
	updateCmd.Flags().BoolVar(&noBackupFlag, "no-backup", false, "Skip safety backup when updating")
	updateCmd.MarkFlagsMutuallyExclusive("backup", "no-backup")
	updateCmd.Flags().BoolVarP(&nonInteract, "yes", "y", false, "Apply updates without interactive confirmation")

	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore files from a timestamped safety backup snapshot",
		RunE:  runRestore,
	}
	restoreCmd.Flags().StringVar(&restoreID, "id", "", "Snapshot ID to restore (defaults to latest matching snapshot)")
	restoreCmd.Flags().StringVarP(&targetType, "target", "t", "", "Filter snapshots by harness (claude, antigravity, zed, opencode)")
	restoreCmd.Flags().BoolVar(&globalInstall, "global", false, "Filter by global scope")
	restoreCmd.Flags().BoolVarP(&nonInteract, "yes", "y", false, "Restore without interactive prompt")

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Safely remove tracked plugin files while preserving your custom skills",
		RunE:  runUninstall,
	}
	uninstallCmd.Flags().StringVarP(&targetType, "target", "t", "", "Target harness to uninstall (claude, antigravity, zed, opencode)")
	uninstallCmd.Flags().BoolVar(&globalInstall, "global", false, "Uninstall from global user-level directory")
	uninstallCmd.Flags().BoolVar(&restoreConfig, "restore-config", false, "Restore pre-install config file from backup if available")
	uninstallCmd.Flags().BoolVarP(&nonInteract, "yes", "y", false, "Uninstall without interactive confirmation")

	initCmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Create a new skill from template",
		Args:  cobra.ExactArgs(1),
		RunE:  runInit,
	}
	var initModel string
	var initTags []string
	var initLangs []string
	var initDesc string
	initCmd.Flags().StringVar(&initModel, "model", "sonnet", "Model to use (haiku, sonnet, opus)")
	initCmd.Flags().StringSliceVar(&initTags, "tag", nil, "Tags for the skill")
	initCmd.Flags().StringSliceVar(&initLangs, "lang", []string{"any"}, "Languages for the skill")
	initCmd.Flags().StringVarP(&initDesc, "desc", "d", "", "Description of the skill")

	rootCmd.AddCommand(versionCmd, listCmd, guideCmd, updateCmd, restoreCmd, uninstallCmd, initCmd)
	return rootCmd
}

func isInteractiveSession(cmd *cobra.Command) bool {
	if cmd == nil || nonInteract || dryRun || fromSource != "" || installMode != "" || targetType != "" || trackName != "" || len(selectedSkills) > 0 {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func renderProjectConfigBytes(baseTemplate []byte, harness string, projectDir string) []byte {
	info := detectProject(projectDir)
	rendered := applyProjectDetection(baseTemplate, info, content)
	return rendered
}

func runInstall(cmd *cobra.Command, args []string) error {
	env := platform.EnvFromOS()
	cat, err := catalog.Load(content)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	applyConfig(cfg)

	if isInteractiveSession(cmd) {
		cwd, _ := os.Getwd()
		m := tui.NewModel(tui.Config{
			Env:                   env,
			Catalog:               cat,
			InitialBackup:         !noBackupFlag && backupFlag,
			ProjectDir:            cwd,
			Version:               version,
			UpdateRunner:          updateRunner,
			ProjectConfigRenderer: renderProjectConfigBytes,
		})
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	mode, err := askInstallMode(reader)
	if err != nil {
		return err
	}

	target, err := getTarget(reader, mode)
	if err != nil {
		return err
	}

	if mode == modeConfigOnly && skipClaude {
		return fmt.Errorf("--mode config-only and --skip-claude-md are contradictory")
	}
	if mode == modeAgentsOnly && skipAgents {
		return fmt.Errorf("--mode agents-only and --skip-agents are contradictory")
	}

	inst := installer.New(content, installer.Options{
		Force:  force,
		DryRun: dryRun,
	})

	switch mode {
	case modeConfigOnly:
		return runConfigOnly(cmd, reader, env, cat, inst, target)
	case modeAgentsOnly:
		return runAgentsOnly(cmd, reader, env, inst, target)
	default:
		return runFullInstall(cmd, reader, env, cat, inst, target)
	}
}

func runFullInstall(cmd *cobra.Command, reader *bufio.Reader, env platform.Env, cat *catalog.Catalog, inst *installer.Installer, target Target) error {
	scope, err := askScope(reader, target)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	paths, err := env.ResolvePaths(target.Harness, scope, cwd)
	if err != nil {
		return err
	}

	if fromSource != "" {
		fmt.Println("\nInstalling skills from external source...")
		var results []string
		if strings.HasPrefix(fromSource, "http://") || strings.HasPrefix(fromSource, "https://") {
			if strings.Contains(fromSource, "github.com") || strings.Contains(fromSource, "gitlab.com") {
				results, err = inst.InstallFromGit(fromSource, paths.SkillsDir)
			} else {
				results, err = inst.InstallFromURL(fromSource, paths.SkillsDir)
			}
		} else {
			results, err = inst.InstallFromLocal(fromSource, paths.SkillsDir)
		}
		if err != nil {
			return err
		}
		for _, r := range results {
			fmt.Println(r)
		}
		return nil
	}

	chosenTrack := trackName
	if chosenTrack == "" {
		if len(selectedSkills) > 0 {
			chosenTrack = "custom"
		} else {
			chosenTrack = "opinionated"
		}
	}
	track, hasTrack := cat.TrackByID(chosenTrack)
	if !hasTrack {
		return fmt.Errorf("unknown track %q (valid: opinionated, basics, product, copy-polish, custom)", chosenTrack)
	}

	doBackup, err := resolveBackupChoice(cmd, reader)
	if err != nil {
		return err
	}

	var allowSkills []string
	if chosenTrack == "custom" {
		allowSkills = selectedSkills
	} else if len(tags) == 0 && len(languages) == 0 {
		allowSkills, err = cat.TrackSkills(chosenTrack, target.Harness)
		if err != nil {
			return err
		}
	}

	updateConfig := false
	if scope == "project" && target.ConfigPath != "" && !skipClaude {
		updateConfig, err = askUpdateConfig(reader, target)
		if err != nil {
			return err
		}
	}

	var candidateTargets []string
	if paths.ConfigFile != "" {
		candidateTargets = append(candidateTargets, paths.ConfigFile)
	}
	if paths.HooksJSON != "" {
		candidateTargets = append(candidateTargets, paths.HooksJSON)
	}
	var backupID string
	if doBackup && !dryRun {
		mgr := backup.NewManager(env.BackupsDir())
		mf, bErr := mgr.CreateSnapshot(target.Harness, scope, chosenTrack, candidateTargets, env.StateFilePath())
		if bErr != nil {
			return bErr
		}
		backupID = mf.ID
	}

	fmt.Println("\nInstalling skills...")
	results, err := inst.InstallSelectedSkillsForHarness(paths.SkillsDir, target.Harness, scope == "global", allowSkills, tags, languages)
	if err != nil {
		return err
	}
	for _, r := range results {
		fmt.Println(r)
	}

	var allActions []string
	allActions = append(allActions, results...)

	if !skipAgents && track.InstallAgents && paths.AgentsDir != "" {
		overwrite, err := askOverwriteAgents(reader, paths.AgentsDir)
		if err != nil {
			return err
		}
		if overwrite {
			agentInst := inst
			if !inst.HasForce() {
				agentInst = installer.New(content, installer.Options{Force: true, DryRun: dryRun})
			}
			fmt.Println("\nInstalling agents...")
			agentResults, err := agentInst.InstallAgentsForHarness(paths.AgentsDir, target.Harness, scope == "global", nil)
			if err != nil {
				return err
			}
			for _, r := range agentResults {
				fmt.Println(r)
			}
			allActions = append(allActions, agentResults...)
		} else {
			fmt.Println("\nSkipping agent installation.")
		}
	}

	hooksMerged := false
	hooksCreated := false
	if scope == "global" && !skipClaude {
		fmt.Println("\nInstalling global harness assets...")
		if target.Harness == installer.HarnessAntigravity {
			globalResults, hm, hc, err := inst.InstallGlobalHarnessAssetsWithOptions(paths.GeminiHome, target.Harness, installer.GlobalAssetOptions{
				LightweightConfig: track.LightweightConfig,
				InstallHooks:      track.InstallHooks,
				Env:               env,
			})
			if err != nil {
				return err
			}
			for _, r := range globalResults {
				fmt.Println(r)
			}
			allActions = append(allActions, globalResults...)
			hooksMerged = hm
			hooksCreated = hc
		} else if paths.ConfigFile != "" {
			tpl, err := cat.GlobalConfigTemplate(target.Harness, track.LightweightConfig)
			if err != nil {
				return err
			}
			if !dryRun {
				if err := os.MkdirAll(filepath.Dir(paths.ConfigFile), 0755); err == nil {
					if _, uErr := installer.UnlinkIfSymlink(paths.ConfigFile, false); uErr == nil {
						if force || !fileExists(paths.ConfigFile) {
							_ = os.WriteFile(paths.ConfigFile, tpl, 0644)
							line := fmt.Sprintf("CREATED: %s", paths.ConfigFile)
							fmt.Println(line)
							allActions = append(allActions, line)
						}
					}
				}
			}
		}
	}

	if !skipCommands && paths.CommandsDir != "" {
		fmt.Println("\nInstalling commands...")
		allowedCmds := cat.CommandsForSkills(target.Harness, allowSkills)
		cmdResults, err := inst.InstallCommandsForHarness(paths.CommandsDir, target.Harness, allowedCmds)
		if err != nil {
			return err
		}
		for _, r := range cmdResults {
			fmt.Println(r)
		}
		allActions = append(allActions, cmdResults...)
	}

	if updateConfig {
		if err := generateConfigFile(inst, target, reader); err != nil {
			fmt.Printf("Warning: Could not generate %s: %v\n", target.ConfigPath, err)
		} else {
			allActions = append(allActions, "CREATED: "+filepath.Join(cwd, target.ConfigPath))
		}
	}

	if scope == "project" && !dryRun && target.Harness == installer.HarnessClaude {
		ensureGitignoreAllowsProjectJSON(reader)
	}

	if !dryRun && target.Harness == installer.HarnessClaude {
		if err := askCodebaseMemoryMCP(reader, env); err != nil {
			fmt.Printf("Warning: Codebase Memory MCP setup skipped: %v\n", err)
		}
	}

	if !dryRun {
		var installedFiles []string
		for _, act := range allActions {
			for _, pfx := range []string{"CREATED: ", "UPDATED: "} {
				if strings.HasPrefix(act, pfx) {
					p := strings.TrimSpace(strings.TrimPrefix(act, pfx))
					if paths.HooksJSON != "" && filepath.Clean(p) == filepath.Clean(paths.HooksJSON) {
						continue
					}
					installedFiles = append(installedFiles, filepath.Clean(p))
				}
			}
		}
		store := state.NewStore(env.StateFilePath())
		_ = store.Upsert(state.Receipt{
			Harness:          target.Harness,
			Scope:            scope,
			RootDir:          paths.RootDir,
			Track:            chosenTrack,
			Skills:           allowSkills,
			InstalledFiles:   installedFiles,
			HooksMerged:      hooksMerged,
			HooksFileCreated: hooksCreated,
			Version:          version,
			LastBackupID:     backupID,
		})
	}

	if dryRun {
		fmt.Println("\n(dry run - no files were modified)")
	} else {
		fmt.Printf("\nDone! %s (%s track) configured for %s.\n", target.Name, track.Name, scope)
		if backupID != "" {
			fmt.Printf("Safety backup saved: %s (restore anytime with `skill-installer restore`)\n", backupID)
		}
		fmt.Printf("New to these commands? Run `skill-installer guide -t %s --track %s` for a 60-second tour!\n", target.Harness, chosenTrack)
	}

	return nil
}

func generateConfigFile(_ *installer.Installer, target Target, reader *bufio.Reader) error {
	configPath := filepath.Join(".", target.ConfigPath)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}
	info := detectProject(cwd)
	if info.Framework != "" {
		fmt.Printf("Detected: %s (%s)\n", info.Framework, info.Name)
	} else {
		fmt.Printf("Detected: unknown project type (using defaults for %s)\n", info.Name)
	}

	if dryRun {
		exists, isSymlink := pathExistsOrSymlink(configPath)
		if exists || isSymlink {
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

	cat, err := catalog.Load(content)
	if err != nil {
		return err
	}
	baseContent, err := cat.ProjectConfigTemplate(target.Harness)
	if err != nil {
		return fmt.Errorf("reading template: %w", err)
	}

	configContent := applyProjectDetection(baseContent, info, content)

	exists, isSymlink := pathExistsOrSymlink(configPath)
	if exists || isSymlink {
		if force {
		} else if nonInteract {
			fmt.Printf("SKIP: %s (already exists, use --force to overwrite)\n", configPath)
			return nil
		} else {
			fmt.Printf("%s already exists. Overwrite? [y/N]: ", target.ConfigPath)
			input, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "y" && input != "yes" {
				fmt.Printf("SKIP: %s\n", configPath)
				return nil
			}
		}
	}

	if _, err := installer.UnlinkIfSymlink(configPath, false); err != nil {
		return err
	}

	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		return err
	}
	fmt.Printf("CREATED: %s\n", configPath)
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
	if trackName == "" && cfg.Track != "" {
		trackName = cfg.Track
	}
	if len(selectedSkills) == 0 && len(cfg.Skills) > 0 {
		selectedSkills = cfg.Skills
	}
	if !backupExplicit && cfg.Backup != nil {
		backupFlag = *cfg.Backup
		noBackupFlag = !*cfg.Backup
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

func getTarget(reader *bufio.Reader, mode string) (Target, error) {
	options := filterTargetsByMode(targetOrder, mode)

	if targetType != "" {
		t, ok := targets[targetType]
		if !ok {
			return Target{}, fmt.Errorf("unknown target %q (supported: claude, antigravity, zed, opencode)", targetType)
		}
		if err := validateTargetForMode(t, mode); err != nil {
			return Target{}, err
		}
		return t, nil
	}

	if nonInteract {
		return targets[installer.HarnessClaude], nil
	}

	prompt, pathFunc := modePromptInfo(mode)
	fmt.Println(prompt)
	fmt.Println()
	for i, key := range options {
		t := targets[key]
		fmt.Printf("  %d) %s (%s)\n", i+1, t.Name, pathFunc(t))
	}
	fmt.Println()
	fmt.Print("Enter choice [1]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return Target{}, err
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return targets[options[0]], nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		return Target{}, fmt.Errorf("invalid choice: %s", input)
	}

	return targets[options[choice-1]], nil
}

func filterTargetsByMode(allOptions []string, mode string) []string {
	var filtered []string
	for _, key := range allOptions {
		t := targets[key]
		switch mode {
		case modeConfigOnly:
			if t.ConfigPath == "" {
				continue
			}
		case modeAgentsOnly:
			if t.AgentsPath == "" {
				continue
			}
		}
		filtered = append(filtered, key)
	}
	return filtered
}

func validateTargetForMode(t Target, mode string) error {
	switch mode {
	case modeConfigOnly:
		if t.ConfigPath == "" {
			return fmt.Errorf("config file generation is not supported for %s", t.Name)
		}
	case modeAgentsOnly:
		if t.AgentsPath == "" {
			return fmt.Errorf("agents are not supported for %s", t.Name)
		}
	}
	return nil
}

func modePromptInfo(mode string) (string, func(Target) string) {
	switch mode {
	case modeConfigOnly:
		return "Where would you like to generate the config file?", func(t Target) string {
			if t.ConfigPath == "" {
				return "auto-detected"
			}
			return t.ConfigPath
		}
	case modeAgentsOnly:
		return "Where would you like to install the agents?", func(t Target) string {
			if t.AgentsPath == "" {
				return "auto-detected"
			}
			return t.AgentsPath
		}
	default:
		return "Where would you like to install the skills?", func(t Target) string {
			if t.SkillsPath == "" {
				return "auto-detected"
			}
			return t.SkillsPath
		}
	}
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

func askScope(reader *bufio.Reader, _ Target) (string, error) {
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

func askInstallMode(reader *bufio.Reader) (string, error) {
	if installMode != "" {
		switch installMode {
		case modeFullInstall, modeConfigOnly, modeAgentsOnly:
			return installMode, nil
		default:
			return "", fmt.Errorf("unknown mode: %s (valid: full, config-only, agents-only)", installMode)
		}
	}

	if nonInteract {
		return modeFullInstall, nil
	}

	fmt.Println("\nWhat would you like to do?")
	fmt.Println("  1) Full installation (skills, agents, commands, and config file)")
	fmt.Println("  2) Generate config file only (e.g., CLAUDE.md, GEMINI.md, AGENTS.md)")
	fmt.Println("  3) Install agents only")
	fmt.Print("Enter choice [1]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)

	switch input {
	case "", "1":
		return modeFullInstall, nil
	case "2":
		return modeConfigOnly, nil
	case "3":
		return modeAgentsOnly, nil
	default:
		return "", fmt.Errorf("invalid choice: %s", input)
	}
}

func runConfigOnly(cmd *cobra.Command, reader *bufio.Reader, env platform.Env, cat *catalog.Catalog, inst *installer.Installer, target Target) error {
	if target.ConfigPath == "" {
		return fmt.Errorf("config file generation is not supported for %s", target.Name)
	}

	_, _ = resolveBackupChoice(cmd, reader)

	if globalInstall {
		paths, err := env.ResolvePaths(target.Harness, "global", ".")
		if err != nil {
			return err
		}
		fmt.Println("\nInstalling global harness configuration...")
		if target.Harness == installer.HarnessAntigravity {
			results, _, _, err := inst.InstallGlobalHarnessAssetsWithOptions(paths.GeminiHome, target.Harness, installer.GlobalAssetOptions{
				InstallHooks: true,
				Env:          env,
			})
			if err != nil {
				return err
			}
			for _, r := range results {
				fmt.Println(r)
			}
		} else {
			tpl, err := cat.GlobalConfigTemplate(target.Harness, false)
			if err != nil {
				return err
			}
			if !dryRun {
				if err := os.MkdirAll(filepath.Dir(paths.ConfigFile), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(paths.ConfigFile, tpl, 0644); err != nil {
					return err
				}
				fmt.Printf("CREATED: %s\n", paths.ConfigFile)
			} else {
				fmt.Printf("WOULD CREATE: %s\n", paths.ConfigFile)
			}
		}
		if dryRun {
			fmt.Println("\n(dry run - no files were modified)")
		} else {
			fmt.Println("\nDone! Global harness configuration installed successfully.")
		}
		return nil
	}

	fmt.Printf("\nGenerating %s...\n", target.ConfigPath)
	if err := generateConfigFile(inst, target, reader); err != nil {
		return fmt.Errorf("could not generate %s: %w", target.ConfigPath, err)
	}

	if dryRun {
		fmt.Println("\n(dry run - no files were modified)")
	} else {
		fmt.Printf("\nDone! %s generated successfully.\n", target.ConfigPath)
	}
	return nil
}

func runAgentsOnly(cmd *cobra.Command, reader *bufio.Reader, env platform.Env, inst *installer.Installer, target Target) error {
	if target.AgentsPath == "" {
		return fmt.Errorf("agents are not supported for %s", target.Name)
	}

	scope, err := askScope(reader, target)
	if err != nil {
		return err
	}

	_, _ = resolveBackupChoice(cmd, reader)

	cwd, _ := os.Getwd()
	paths, err := env.ResolvePaths(target.Harness, scope, cwd)
	if err != nil {
		return err
	}
	if paths.AgentsDir == "" {
		return fmt.Errorf("agents are not supported for %s in %s scope", target.Name, scope)
	}

	overwrite, err := askOverwriteAgents(reader, paths.AgentsDir)
	if err != nil {
		return err
	}

	agentInst := inst
	if overwrite && !inst.HasForce() {
		agentInst = installer.New(content, installer.Options{Force: true, DryRun: dryRun})
	}

	if !overwrite {
		fmt.Println("\nSkipping agent installation.")
	} else {
		fmt.Println("\nInstalling agents...")
		agentResults, err := agentInst.InstallAgentsForHarness(paths.AgentsDir, target.Harness, scope == "global", nil)
		if err != nil {
			return err
		}
		for _, r := range agentResults {
			fmt.Println(r)
		}
	}

	if dryRun {
		fmt.Println("\n(dry run - no files were modified)")
	} else if overwrite {
		fmt.Println("\nDone! Agents installed successfully.")
	}
	return nil
}

func askOverwriteAgents(reader *bufio.Reader, agentsDest string) (bool, error) {
	if info, err := os.Lstat(agentsDest); err == nil && info.Mode()&os.ModeSymlink != 0 {
		if force || nonInteract {
			return true, nil
		}
		fmt.Printf("\n%s is a symlink. Replace symlink with installed agent files? [y/N]: ", agentsDest)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		input = strings.TrimSpace(strings.ToLower(input))
		return input == "y" || input == "yes", nil
	}

	existing, _ := filepath.Glob(filepath.Join(agentsDest, "*.md"))
	if len(existing) == 0 {
		return true, nil
	}

	if force {
		return true, nil
	}

	if nonInteract {
		return false, nil
	}

	fmt.Printf("\nExisting agent files found in %s:\n", agentsDest)
	for _, f := range existing {
		fmt.Printf("  - %s\n", filepath.Base(f))
	}
	fmt.Print("Overwrite existing agent files? [y/N]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}

func ensureGitignoreAllowsProjectJSON(reader *bufio.Reader) {
	gitignorePath := ".gitignore"
	info, err := os.Lstat(gitignorePath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		return
	}
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return
	}

	contentStr := string(data)
	lines := strings.Split(contentStr, "\n")

	hasClaudeGlob := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".claude/*" || trimmed == ".claude/" {
			hasClaudeGlob = true
			break
		}
	}
	if !hasClaudeGlob {
		return
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "!.claude/project.json" {
			return
		}
	}

	if !nonInteract {
		fmt.Println("\n.gitignore blocks .claude/* but .claude/project.json needs to be tracked")
		fmt.Println("for project initialization state to persist across sessions.")
		fmt.Print("Add !.claude/project.json to .gitignore? [Y/n]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "" && input != "y" && input != "yes" {
			fmt.Println("Skipped. You can add !.claude/project.json to .gitignore manually.")
			return
		}
	}

	var result []string
	inserted := false
	for _, line := range lines {
		result = append(result, line)
		trimmed := strings.TrimSpace(line)
		if !inserted && (trimmed == ".claude/*" || trimmed == ".claude/") {
			result = append(result, "!.claude/project.json")
			inserted = true
		}
	}

	if !inserted {
		return
	}

	if err := os.WriteFile(gitignorePath, []byte(strings.Join(result, "\n")), 0644); err != nil {
		fmt.Printf("Warning: could not update .gitignore: %v\n", err)
		return
	}
	fmt.Println("UPDATED: .gitignore (added !.claude/project.json)")
}

func runList(cmd *cobra.Command, args []string) error {
	inst := installer.New(content, installer.Options{})
	cat, err := catalog.Load(content)
	if err != nil {
		return err
	}

	harness := installer.HarnessClaude
	if targetType != "" {
		t, ok := targets[targetType]
		if !ok {
			return fmt.Errorf("unknown target: %s", targetType)
		}
		harness = t.Harness
	}

	var trackAllow map[string]bool
	if trackName != "" {
		ts, err := cat.TrackSkills(trackName, harness)
		if err != nil {
			return err
		}
		trackAllow = make(map[string]bool, len(ts))
		for _, s := range ts {
			trackAllow[s] = true
		}
	}

	skills, err := inst.ListSkillsForHarness(harness)
	if err != nil {
		return err
	}

	var matchedNames []string
	for _, s := range skills {
		if trackAllow != nil && !trackAllow[s.Name] {
			continue
		}
		if !installer.MatchesFilter(s, tags, languages) {
			continue
		}
		matchedNames = append(matchedNames, s.Name)
	}

	if len(matchedNames) == 0 {
		fmt.Println("No skills match the specified filters.")
		return nil
	}

	fmt.Printf("Curated tracks for %s:\n", tui.HarnessDisplayName(harness))
	for _, tr := range cat.Tracks {
		ts, _ := cat.TrackSkills(tr.ID, harness)
		fmt.Printf("  • %-14s %-28s (%d skills) — %s\n", tr.ID, tr.Name, len(ts), tr.Tagline)
	}
	fmt.Printf("\nAvailable skills (%d):\n\n", len(matchedNames))
	fmt.Print(tui.RenderSkillTable(cat, harness, trackName, listLong, matchedNames))
	return nil
}

func runGuide(cmd *cobra.Command, args []string) error {
	env := platform.EnvFromOS()
	cat, err := catalog.Load(content)
	if err != nil {
		return err
	}
	harness := targetType
	if harness == "" {
		harness = installer.HarnessClaude
	}
	if !catalog.IsSupportedHarness(harness) {
		return fmt.Errorf("unsupported target %q (supported: claude, antigravity, zed, opencode)", harness)
	}
	tr := trackName
	if tr == "" {
		tr = "opinionated"
	}

	if !nonInteract && cmd != nil && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		cwd, _ := os.Getwd()
		m := tui.NewModel(tui.Config{
			Env:            env,
			Catalog:        cat,
			InitialHarness: harness,
			InitialTrack:   tr,
			StartInGuide:   true,
			ProjectDir:     cwd,
			Version:        version,
		})
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err := p.Run()
		return err
	}

	fmt.Print(tui.RenderPlainGuide(cat, harness, tr))
	return nil
}

func resolveReceiptForOperation(env platform.Env, cat *catalog.Catalog) (*state.Receipt, error) {
	store := state.NewStore(env.StateFilePath())
	cwd, _ := os.Getwd()
	scope := "project"
	if globalInstall {
		scope = "global"
	}

	if targetType != "" {
		if !catalog.IsSupportedHarness(targetType) {
			return nil, fmt.Errorf("unsupported target %q (supported: claude, antigravity, zed, opencode)", targetType)
		}
		paths, err := env.ResolvePaths(targetType, scope, cwd)
		if err != nil {
			return nil, err
		}
		if rec, ok, err := store.Find(targetType, scope, paths.RootDir); err == nil && ok {
			return rec, nil
		}
		if leg, ok := state.DiscoverLegacyReceipt(env, cat, targetType, scope, cwd); ok {
			return leg, nil
		}
		return &state.Receipt{
			Harness: targetType,
			Scope:   scope,
			RootDir: paths.RootDir,
			Track:   "opinionated",
		}, nil
	}

	receipts, _ := store.Load()
	if len(receipts) > 0 {
		return &receipts[0], nil
	}
	for _, h := range targetOrder {
		if leg, ok := state.DiscoverLegacyReceipt(env, cat, h, scope, cwd); ok {
			return leg, nil
		}
	}
	paths, _ := env.ResolvePaths(installer.HarnessClaude, scope, cwd)
	return &state.Receipt{
		Harness: installer.HarnessClaude,
		Scope:   scope,
		RootDir: paths.RootDir,
		Track:   "opinionated",
	}, nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	env := platform.EnvFromOS()
	cat, err := catalog.Load(content)
	if err != nil {
		return err
	}
	rec, err := resolveReceiptForOperation(env, cat)
	if err != nil {
		return err
	}

	rep, err := updater.Check(context.Background(), updateRunner, *rec, version)
	if err != nil {
		return err
	}

	if updateJSON {
		data, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Checking updates for %s (%s · track: %s)...\n", tui.HarnessDisplayName(rec.Harness), rec.Scope, rec.Track)
	if !rep.UpdateAvailable && !rep.BinaryUpdate {
		fmt.Printf("Already up to date on main (%s).\n", rep.LatestSHA)
		return nil
	}

	if rep.UpdateAvailable {
		fmt.Printf("Skill/catalog update available on main: %s -> %s\n", rep.InstalledSHA, rep.LatestSHA)
		if len(rep.ChangedSkills) > 0 {
			fmt.Printf("Changed skills: %s\n", strings.Join(rep.ChangedSkills, ", "))
		}
		for _, c := range rep.Commits {
			fmt.Printf("  • %s %s\n", c.SHA, c.Message)
		}
	}
	if rep.BinaryUpdate {
		fmt.Printf("New CLI release available: v%s -> v%s (run `skill-installer update --self`)\n", rep.CurrentVersion, rep.LatestVersion)
	}

	if updateCheck && !updateApply && !updateSelf {
		return nil
	}

	if updateSelf && rep.LatestVersion != "" {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		upgraded, err := updater.SelfUpgradeBinary(context.Background(), env, exePath, rep.LatestVersion, nil, "")
		if err != nil {
			return err
		}
		fmt.Printf("Upgraded skill-installer binary to v%s.\n", upgraded)
		if !updateApply {
			return nil
		}
	}

	reader := bufio.NewReader(os.Stdin)
	if !nonInteract && !updateApply {
		fmt.Print("\nApply skill updates from main now? [Y/n]: ")
		in, _ := reader.ReadString('\n')
		trimmed := strings.TrimSpace(strings.ToLower(in))
		if trimmed == "n" || trimmed == "no" {
			fmt.Println("Update skipped.")
			return nil
		}
	}

	doBackup, err := resolveBackupChoice(cmd, reader)
	if err != nil {
		return err
	}

	res, newSHA, err := updater.Apply(context.Background(), updateRunner, env, *rec, updater.ApplyOptions{
		Backup:                doBackup,
		Force:                 true,
		CurrentVersion:        version,
		ProjectConfigRenderer: renderProjectConfigBytes,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Updated %s (%s) to commit %s (%d skills synced).\n", tui.HarnessDisplayName(rec.Harness), rec.Scope, newSHA, len(res.Skills))
	if res.BackupID != "" {
		fmt.Printf("Safety backup saved: %s\n", res.BackupID)
	}
	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	env := platform.EnvFromOS()
	mgr := backup.NewManager(env.BackupsDir())

	chosenID := restoreID
	if chosenID == "" {
		snaps, err := mgr.ListSnapshots(targetType)
		if err != nil {
			return err
		}
		if len(snaps) == 0 {
			return fmt.Errorf("no safety backup snapshots found in %s", env.BackupsDir())
		}
		chosenID = snaps[0].ID
	}

	res, err := mgr.RestoreSnapshot(chosenID, env.StateFilePath())
	if err != nil {
		return err
	}
	fmt.Printf("Restored snapshot %s (%d files restored, %d created files removed).\n",
		res.SnapshotID, len(res.RestoredFiles), len(res.RemovedFiles))
	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	env := platform.EnvFromOS()
	cat, err := catalog.Load(content)
	if err != nil {
		return err
	}
	rec, err := resolveReceiptForOperation(env, cat)
	if err != nil {
		return err
	}

	res, err := state.Uninstall(env, cat, *rec, backup.NewManager(env.BackupsDir()), restoreConfig)
	if err != nil {
		return err
	}
	store := state.NewStore(env.StateFilePath())
	_ = store.Remove(rec.Harness, rec.Scope, rec.RootDir)

	fmt.Printf("Uninstalled %s (%s): removed %d tracked files.\n",
		tui.HarnessDisplayName(rec.Harness), rec.Scope, len(res.RemovedFiles))
	if res.RestoredConfig != "" {
		fmt.Printf("Restored pre-install config: %s\n", res.RestoredConfig)
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

	exists, isSymlink := pathExistsOrSymlink(filename)
	_, dirIsSymlink := pathExistsOrSymlink(skillDir)
	if (exists || isSymlink || dirIsSymlink) && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", filename)
	}

	if dryRun {
		fmt.Printf("WOULD CREATE: %s\n", filename)
		return nil
	}

	if _, err := installer.UnlinkIfSymlink(skillDir, false); err != nil {
		return err
	}
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", skillDir, err)
	}
	if _, err := installer.UnlinkIfSymlink(filename, false); err != nil {
		return err
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

func pathExistsOrSymlink(path string) (bool, bool) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true, true
	}
	return !info.IsDir(), false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func askCodebaseMemoryMCP(reader *bufio.Reader, env platform.Env) error {
	if nonInteract {
		return nil
	}

	binName := cbMemBinName
	if env.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(env.HomeDir, ".local", "bin", binName)

	binaryExists := fileExists(binPath)
	configExists := mcpServerConfigured(cbMemBinName, env.HomeDir)
	if binaryExists && configExists {
		fmt.Printf("\nCodebase Memory MCP already installed at %s — skipping.\n", binPath)
		return nil
	}
	if configExists {
		fmt.Println("\nCodebase Memory MCP server already configured — skipping.")
		return nil
	}

	if binaryExists {
		fmt.Printf("\nCodebase Memory MCP binary found at %s but MCP config is missing.\n", binPath)
		fmt.Print("Would you like to configure it now? [Y/n]: ")
	} else {
		fmt.Print("\nWould you like to install the Codebase Memory MCP server for codebase indexing and search? [Y/n]: ")
	}
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "n" || input == "no" {
		return nil
	}

	fmt.Println("\nWhere should the MCP server be configured?")
	fmt.Println("  1) Global — available to all projects (recommended)")
	fmt.Println("  2) Local — this project only (.mcp.json)")
	fmt.Print("Enter choice [1]: ")
	scopeInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	scopeInput = strings.TrimSpace(scopeInput)
	isGlobal := scopeInput != "2"

	if !binaryExists {
		plat := cbMemPlatform(env)
		if plat == "" {
			return fmt.Errorf("unsupported platform: %s/%s — install manually from https://github.com/%s/releases", env.GOOS, env.GOARCH, cbMemRepo)
		}

		ext := "tar.gz"
		if env.GOOS == "windows" {
			ext = "zip"
		}
		archiveName := fmt.Sprintf("codebase-memory-mcp-%s.%s", plat, ext)
		downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", cbMemRepo, cbMemVersion, archiveName)
		checksumsURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", cbMemRepo, cbMemVersion)

		fmt.Printf("\nDownloading codebase-memory-mcp %s (%s)...\n", cbMemVersion, plat)

		checksumsData, err := httpGetBytes(checksumsURL)
		if err != nil {
			return fmt.Errorf("downloading checksums: %w", err)
		}

		expectedHash := findChecksumForFile(string(checksumsData), archiveName)
		if expectedHash == "" {
			return fmt.Errorf("checksum not found for %s in checksums.txt", archiveName)
		}

		tmpDir, err := os.MkdirTemp("", "cbmem-*")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Join(tmpDir, archiveName)
		if err := httpDownloadFile(downloadURL, archivePath); err != nil {
			return fmt.Errorf("downloading %s: %w", archiveName, err)
		}

		actualHash, err := sha256File(archivePath)
		if err != nil {
			return fmt.Errorf("computing checksum: %w", err)
		}
		if actualHash != expectedHash {
			return fmt.Errorf("checksum verification failed!\n  expected: %s\n  got:      %s", expectedHash, actualHash)
		}
		fmt.Println("Checksum verified.")

		if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(binPath), err)
		}

		if ext == "zip" {
			err = extractZipBinary(archivePath, binName, binPath)
		} else {
			err = extractTarGzBinary(archivePath, binName, binPath)
		}
		if err != nil {
			return fmt.Errorf("extracting binary: %w", err)
		}

		if err := os.Chmod(binPath, 0700); err != nil {
			return fmt.Errorf("setting permissions: %w", err)
		}
		fmt.Printf("Installed: %s\n", binPath)
	}

	serverConfig := map[string]interface{}{
		"type":    "stdio",
		"command": binPath,
	}

	if isGlobal {
		if err := addGlobalMCPServer("codebase-memory-mcp", serverConfig, env.HomeDir); err != nil {
			return fmt.Errorf("configuring global MCP: %w", err)
		}
		fmt.Println("Configured: ~/.claude/settings.json (global)")
	} else {
		if err := addLocalMCPServer("codebase-memory-mcp", serverConfig); err != nil {
			return fmt.Errorf("configuring local MCP: %w", err)
		}
		fmt.Println("Configured: .mcp.json (local)")
	}

	return nil
}

func cbMemPlatform(env platform.Env) string {
	switch {
	case env.GOOS == "darwin" && env.GOARCH == "arm64":
		return "darwin-arm64"
	case env.GOOS == "darwin" && env.GOARCH == "amd64":
		return "darwin-amd64"
	case env.GOOS == "linux" && env.GOARCH == "amd64":
		return "linux-amd64"
	case env.GOOS == "linux" && env.GOARCH == "arm64":
		return "linux-arm64"
	case env.GOOS == "windows" && env.GOARCH == "amd64":
		return "windows-amd64"
	default:
		return ""
	}
}

func mcpServerConfigured(name, home string) bool {
	globalPath := filepath.Join(home, ".claude", "settings.json")
	if checkMCPConfigFile(globalPath, name) {
		return true
	}
	return checkMCPConfigFile(".mcp.json", name)
}

func checkMCPConfigFile(path, serverName string) bool {
	if !fileExists(path) {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false
	}
	servers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		return false
	}
	_, exists := servers[serverName]
	return exists
}

const allowedDownloadHost = "github.com"

func validateDownloadURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed, got %s", u.Scheme)
	}
	if u.Host != allowedDownloadHost {
		return fmt.Errorf("downloads are restricted to %s, got %s", allowedDownloadHost, u.Host)
	}
	return nil
}

func httpGetBytes(fetchURL string) ([]byte, error) {
	if err := validateDownloadURL(fetchURL); err != nil {
		return nil, err
	}
	resp, err := http.Get(fetchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, fetchURL)
	}
	return io.ReadAll(resp.Body)
}

func httpDownloadFile(fetchURL, destPath string) error {
	if err := validateDownloadURL(fetchURL); err != nil {
		return err
	}
	destPath = filepath.Clean(destPath)
	resp, err := http.Get(fetchURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func sha256File(path string) (string, error) {
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func findChecksumForFile(checksums, filename string) string {
	for _, line := range strings.Split(checksums, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			return parts[0]
		}
	}
	return ""
}

func extractTarGzBinary(archivePath, binName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filepath.Base(hdr.Name) == binName && hdr.Typeflag == tar.TypeReg {
			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, io.LimitReader(tr, 100*1024*1024))
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
	}
	return fmt.Errorf("%s not found in archive", binName)
}

func extractZipBinary(archivePath, binName, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, zf := range r.File {
		if filepath.Base(zf.Name) == binName {
			rc, err := zf.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.Create(destPath)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, io.LimitReader(rc, 100*1024*1024))
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
	}
	return fmt.Errorf("%s not found in archive", binName)
}

func addGlobalMCPServer(name string, serverConfig map[string]interface{}, home string) error {
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	var settings map[string]interface{}
	if fileExists(settingsPath) {
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", settingsPath, err)
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, err)
		}
	} else {
		settings = map[string]interface{}{}
	}

	servers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		servers = map[string]interface{}{}
	}
	servers[name] = serverConfig
	settings["mcpServers"] = servers

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(data, '\n'), 0600)
}

func addLocalMCPServer(name string, serverConfig map[string]interface{}) error {
	mcpPath := ".mcp.json"

	var mcpConfig map[string]interface{}
	if fileExists(mcpPath) {
		data, err := os.ReadFile(mcpPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", mcpPath, err)
		}
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			return fmt.Errorf("parsing %s: %w", mcpPath, err)
		}
	} else {
		mcpConfig = map[string]interface{}{}
	}

	servers, ok := mcpConfig["mcpServers"].(map[string]interface{})
	if !ok {
		servers = map[string]interface{}{}
	}
	servers[name] = serverConfig
	mcpConfig["mcpServers"] = servers

	data, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(mcpPath, append(data, '\n'), 0600)
}
