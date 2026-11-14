package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/backup"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/installer"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/state"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/updater"
)

type screenID int

const (
	screenHome screenID = iota
	screenHarness
	screenTrack
	screenSkillTable
	screenBackup
	screenDone
	screenGuide
	screenUpdate
	screenRestore
	screenUninstall
)

type tickMsg time.Time

type Config struct {
	Env                   platform.Env
	Catalog               *catalog.Catalog
	InitialHarness        string
	InitialScope          string
	InitialTrack          string
	InitialBackup         bool
	StartInGuide          bool
	ProjectDir            string
	Version               string
	UpdateRunner          updater.Runner
	ProjectConfigRenderer func(base []byte, harness string, projectDir string) []byte
}

type Model struct {
	cfg               Config
	screen            screenID
	prevScreen        screenID
	width             int
	height            int
	botFrame          int
	homeIdx           int
	harnessIdx        int
	scope             string
	trackIdx          int
	backupIdx         int
	backupTouched     bool
	doneIdx           int
	guideTab          int
	skillCursor       int
	skillExpanded     map[string]bool
	customSelected    map[string]bool
	showAllInGuide    bool
	installRes        *installer.InstallResult
	statusMsg         string
	errMsg            string
	updateIdx         int
	updateReport      *updater.CheckReport
	snapshots         []backup.Manifest
	restoreIdx        int
	confirmRestore    bool
	receipts          []state.Receipt
	uninstallIdx      int
	confirmUninstall  bool
	editingProjectDir bool
	projectDirInput   string
	projectDirErr     string
}

var harnessOrder = []string{
	catalog.HarnessAntigravity,
	catalog.HarnessClaude,
	catalog.HarnessZed,
	catalog.HarnessOpenCode,
}

var harnessTaglines = map[string]string{
	catalog.HarnessAntigravity: "Configures GEMINI.md, native /slash skills, review subagents, and context-saving hooks",
	catalog.HarnessClaude:      "Adds CLAUDE.md, auto-triggered skills, review subagents, and /start, /shape, /write-a-prd commands",
	catalog.HarnessZed:         "Configures AGENTS.md and .agents/skills (automatically exposed as /slash commands in Zed)",
	catalog.HarnessOpenCode:    "Sets up AGENTS.md, .opencode/skills, review subagents, and .opencode/commands",
}

func NewModel(cfg Config) Model {
	if cfg.InitialScope == "" {
		cfg.InitialScope = "global"
	}
	if cfg.InitialTrack == "" {
		cfg.InitialTrack = "opinionated"
	}
	if cfg.ProjectDir == "" {
		cfg.ProjectDir = "."
	}

	hIdx := 0
	for i, h := range harnessOrder {
		if h == cfg.InitialHarness {
			hIdx = i
			break
		}
	}

	tIdx := 0
	if cfg.Catalog != nil {
		for i, tr := range cfg.Catalog.Tracks {
			if tr.ID == cfg.InitialTrack {
				tIdx = i
				break
			}
		}
	}

	bIdx := 0
	if !cfg.InitialBackup {
		bIdx = 1
	}

	customSel := map[string]bool{
		"plugin-updater": true,
	}

	startScreen := screenHome
	if cfg.StartInGuide {
		startScreen = screenGuide
	} else if cfg.InitialHarness != "" {
		startScreen = screenTrack
	}

	return Model{
		cfg:            cfg,
		screen:         startScreen,
		prevScreen:     screenHome,
		harnessIdx:     hIdx,
		scope:          cfg.InitialScope,
		trackIdx:       tIdx,
		backupIdx:      bIdx,
		skillExpanded:  make(map[string]bool),
		customSelected: customSel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(900*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) selectedHarness() string {
	if m.harnessIdx >= 0 && m.harnessIdx < len(harnessOrder) {
		return harnessOrder[m.harnessIdx]
	}
	return catalog.HarnessAntigravity
}

func (m Model) selectedTrack() catalog.Track {
	if m.cfg.Catalog != nil && m.trackIdx >= 0 && m.trackIdx < len(m.cfg.Catalog.Tracks) {
		return m.cfg.Catalog.Tracks[m.trackIdx]
	}
	return catalog.Track{ID: "opinionated", Name: "Opinionated Dev Workflow"}
}

func (m Model) visibleSkills() []catalog.SkillMetadata {
	if m.cfg.Catalog == nil {
		return nil
	}
	h := m.selectedHarness()
	tr := m.selectedTrack()

	if m.screen == screenGuide {
		if m.showAllInGuide {
			return m.cfg.Catalog.SkillsForHarness(h)
		}
		return m.installedOrSelectedSkills()
	}
	if tr.ID == "custom" {
		return m.cfg.Catalog.SkillsForHarness(h)
	}
	names, err := m.cfg.Catalog.TrackSkills(tr.ID, h)
	if err != nil {
		return m.cfg.Catalog.SkillsForHarness(h)
	}
	var out []catalog.SkillMetadata
	for _, name := range names {
		if sm, ok := m.cfg.Catalog.SkillMeta(name); ok {
			out = append(out, sm)
		}
	}
	return out
}

func (m Model) installedOrSelectedSkills() []catalog.SkillMetadata {
	if m.cfg.Catalog == nil {
		return nil
	}
	h := m.selectedHarness()
	if m.installRes != nil && len(m.installRes.Skills) > 0 {
		var out []catalog.SkillMetadata
		for _, name := range m.installRes.Skills {
			if sm, ok := m.cfg.Catalog.SkillMeta(name); ok {
				out = append(out, sm)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	tr := m.selectedTrack()
	if tr.ID == "custom" {
		var out []catalog.SkillMetadata
		for _, sm := range m.cfg.Catalog.SkillsForHarness(h) {
			if m.customSelected[sm.Name] {
				out = append(out, sm)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	names, err := m.cfg.Catalog.TrackSkills(tr.ID, h)
	if err != nil {
		return m.cfg.Catalog.SkillsForHarness(h)
	}
	var out []catalog.SkillMetadata
	for _, name := range names {
		if sm, ok := m.cfg.Catalog.SkillMeta(name); ok {
			out = append(out, sm)
		}
	}
	return out
}

func (m Model) selectedCustomCount() int {
	if m.cfg.Catalog == nil {
		count := 0
		for _, v := range m.customSelected {
			if v {
				count++
			}
		}
		return count
	}
	count := 0
	for _, sm := range m.cfg.Catalog.SkillsForHarness(m.selectedHarness()) {
		if m.customSelected[sm.Name] {
			count++
		}
	}
	return count
}

func (m Model) existingConfigPresent() bool {
	paths, err := m.cfg.Env.ResolvePaths(m.selectedHarness(), m.scope, m.cfg.ProjectDir)
	if err != nil || paths.ConfigFile == "" {
		return false
	}
	info, err := os.Lstat(paths.ConfigFile)
	return err == nil && !info.IsDir()
}

func (m *Model) clampBackupIdx() {
	maxIdx := 1
	if m.existingConfigPresent() {
		maxIdx = 3
	}
	if !m.cfg.InitialBackup && !m.backupTouched && m.backupIdx == 1 && m.existingConfigPresent() {
		m.backupIdx = 3
	}
	if m.backupIdx > maxIdx {
		m.backupIdx = 0
	}
}

func (m Model) InstallResult() *installer.InstallResult {
	return m.installRes
}

func (m Model) StarterPrompts(max int) []catalog.SkillMetadata {
	all := m.installedOrSelectedSkills()
	if len(all) == 0 || max <= 0 {
		return nil
	}
	hasStart := false
	for _, sm := range all {
		if sm.Name == "start" {
			hasStart = true
			break
		}
	}
	var primary []catalog.SkillMetadata
	var updaterFallback []catalog.SkillMetadata
	for _, sm := range all {
		if sm.StarterPrompt == "" {
			continue
		}
		if hasStart && sm.Name == "begin" {
			continue
		}
		if sm.Name == "plugin-updater" {
			updaterFallback = append(updaterFallback, sm)
			continue
		}
		primary = append(primary, sm)
	}
	combined := append(primary, updaterFallback...)
	if len(combined) > max {
		return combined[:max]
	}
	return combined
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		m.botFrame = (m.botFrame + 1) % len(botFrames)
		return m, tea.Tick(2200*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	case tea.KeyMsg:
		if m.screen == screenHarness && m.editingProjectDir {
			return m.updateHarnessEditProjectDir(msg)
		}
		key := msg.String()
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		if key == "?" && m.screen != screenGuide {
			m.prevScreen = m.screen
			m.screen = screenGuide
			m.skillCursor = 0
			return m, nil
		}

		switch m.screen {
		case screenHome:
			return m.updateHome(key)
		case screenHarness:
			return m.updateHarness(key)
		case screenTrack:
			return m.updateTrack(key)
		case screenSkillTable:
			return m.updateSkillTable(key)
		case screenBackup:
			return m.updateBackup(key)
		case screenDone:
			return m.updateDone(key)
		case screenGuide:
			return m.updateGuide(key)
		case screenUpdate:
			return m.updateUpdateScreen(key)
		case screenRestore:
			return m.updateRestoreScreen(key)
		case screenUninstall:
			return m.updateUninstallScreen(key)
		}
	}
	return m, nil
}

func (m Model) updateHome(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.homeIdx > 0 {
			m.homeIdx--
		}
	case "down", "j":
		if m.homeIdx < 4 {
			m.homeIdx++
		}
	case "enter":
		m.errMsg = ""
		m.statusMsg = ""
		switch m.homeIdx {
		case 0:
			m.screen = screenHarness
		case 1:
			m.prevScreen = screenHome
			m.screen = screenGuide
		case 2:
			m.loadReceipts()
			m.updateIdx = 0
			m.loadUpdateReport()
			m.screen = screenUpdate
		case 3:
			m.loadSnapshots()
			m.screen = screenRestore
		case 4:
			m.loadReceipts()
			m.screen = screenUninstall
		}
	}
	return m, nil
}

func (m Model) updateHarness(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "backspace":
		m.screen = screenHome
	case "up", "k":
		if m.harnessIdx > 0 {
			m.harnessIdx--
		}
	case "down", "j":
		if m.harnessIdx < len(harnessOrder)-1 {
			m.harnessIdx++
		}
	case "tab", "s":
		if m.scope == "global" {
			m.scope = "project"
		} else {
			m.scope = "global"
		}
	case "e", "/":
		m.editingProjectDir = true
		m.projectDirInput = m.cfg.ProjectDir
		m.projectDirErr = ""
	case "enter":
		m.screen = screenTrack
	}
	return m, nil
}

func (m Model) updateHarnessEditProjectDir(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.editingProjectDir = false
		m.projectDirErr = ""
		return m, nil
	case tea.KeyBackspace, tea.KeyDelete:
		runes := []rune(m.projectDirInput)
		if len(runes) > 0 {
			m.projectDirInput = string(runes[:len(runes)-1])
		}
		m.projectDirErr = ""
		return m, nil
	case tea.KeyCtrlU:
		m.projectDirInput = ""
		m.projectDirErr = ""
		return m, nil
	case tea.KeyEnter:
		if strings.TrimSpace(m.projectDirInput) == "" {
			m.projectDirErr = "Please enter a valid project folder path."
			return m, nil
		}
		baseDir, err := os.Getwd()
		if err != nil || baseDir == "" {
			baseDir = m.cfg.ProjectDir
		}
		resolved := m.cfg.Env.ExpandPath(m.projectDirInput, baseDir)
		if resolved == "" {
			m.projectDirErr = "Please enter a valid project folder path."
			return m, nil
		}
		if err := backup.RejectSymlinkInChain(resolved); err != nil {
			m.projectDirErr = fmt.Sprintf("Cannot use symlinked path: %v", err)
			return m, nil
		}
		if linfo, lErr := os.Lstat(resolved); lErr == nil && linfo.Mode()&os.ModeSymlink != 0 {
			m.projectDirErr = fmt.Sprintf("Cannot use symlinked project folder: %s", resolved)
			return m, nil
		}
		if info, statErr := os.Stat(resolved); statErr == nil && !info.IsDir() {
			m.projectDirErr = "That path points to a file — please enter a folder path."
			return m, nil
		}
		m.cfg.ProjectDir = resolved
		m.scope = "project"
		m.editingProjectDir = false
		m.projectDirErr = ""
		return m, nil
	case tea.KeySpace:
		m.projectDirInput += " "
		m.projectDirErr = ""
		return m, nil
	case tea.KeyRunes:
		m.projectDirInput += string(msg.Runes)
		m.projectDirErr = ""
		return m, nil
	}
	return m, nil
}

func (m Model) updateTrack(key string) (tea.Model, tea.Cmd) {
	maxIdx := 0
	if m.cfg.Catalog != nil && len(m.cfg.Catalog.Tracks) > 0 {
		maxIdx = len(m.cfg.Catalog.Tracks) - 1
	}
	switch key {
	case "esc", "backspace":
		m.screen = screenHarness
	case "up", "k":
		if m.trackIdx > 0 {
			m.trackIdx--
		}
	case "down", "j":
		if m.trackIdx < maxIdx {
			m.trackIdx++
		}
	case " ", "p":
		m.skillCursor = 0
		m.screen = screenSkillTable
	case "enter":
		tr := m.selectedTrack()
		if tr.ID == "custom" {
			m.skillCursor = 0
			m.screen = screenSkillTable
			return m, nil
		}
		m.clampBackupIdx()
		m.screen = screenBackup
	}
	return m, nil
}

func (m Model) updateSkillTable(key string) (tea.Model, tea.Cmd) {
	skills := m.visibleSkills()
	tr := m.selectedTrack()
	switch key {
	case "esc", "backspace":
		m.screen = screenTrack
	case "up", "k":
		if m.skillCursor > 0 {
			m.skillCursor--
		}
	case "down", "j":
		if m.skillCursor < len(skills)-1 {
			m.skillCursor++
		}
	case " ":
		if tr.ID == "custom" && m.skillCursor >= 0 && m.skillCursor < len(skills) {
			name := skills[m.skillCursor].Name
			if m.customSelected[name] {
				delete(m.customSelected, name)
			} else {
				m.customSelected[name] = true
			}
		}
	case "d", "enter":
		if key == "enter" && tr.ID == "custom" {
			m.clampBackupIdx()
			m.screen = screenBackup
			return m, nil
		}
		if m.skillCursor >= 0 && m.skillCursor < len(skills) {
			name := skills[m.skillCursor].Name
			m.skillExpanded[name] = !m.skillExpanded[name]
		}
	case "c":
		m.clampBackupIdx()
		m.screen = screenBackup
	}
	return m, nil
}

func (m Model) updateBackup(key string) (tea.Model, tea.Cmd) {
	maxIdx := 1
	if m.existingConfigPresent() {
		maxIdx = 3
	}
	if m.backupIdx > maxIdx {
		m.backupIdx = 0
	}
	switch key {
	case "esc", "backspace":
		m.screen = screenTrack
	case "up", "k", "left":
		if m.backupIdx > 0 {
			m.backupIdx--
			m.backupTouched = true
		}
	case "down", "j", "right":
		if m.backupIdx < maxIdx {
			m.backupIdx++
			m.backupTouched = true
		}
	case "enter":
		m.runInstallation()
		m.screen = screenDone
	}
	return m, nil
}

func (m *Model) runInstallation() {
	if m.cfg.Catalog == nil {
		m.errMsg = "catalog not loaded"
		return
	}
	h := m.selectedHarness()
	tr := m.selectedTrack()
	var customSkills []string
	if tr.ID == "custom" {
		for _, sm := range m.cfg.Catalog.SkillsForHarness(h) {
			if m.customSelected[sm.Name] {
				customSkills = append(customSkills, sm.Name)
			}
		}
	}

	hasExisting := m.existingConfigPresent()
	cfgMode := installer.ConfigModeReplace
	doBackup := m.backupIdx == 0
	if hasExisting {
		switch m.backupIdx {
		case 0:
			cfgMode = installer.ConfigModeMerge
			doBackup = true
		case 1:
			cfgMode = installer.ConfigModeReplace
			doBackup = true
		case 2:
			cfgMode = installer.ConfigModeSkip
			doBackup = true
		case 3:
			cfgMode = installer.ConfigModeReplace
			doBackup = false
		}
	}

	inst := installer.New(m.cfg.Catalog.FS(), installer.Options{Force: true})
	res, err := inst.ExecutePlan(installer.InstallPlan{
		Env:                   m.cfg.Env,
		Catalog:               m.cfg.Catalog,
		Harness:               h,
		Scope:                 m.scope,
		ProjectDir:            m.cfg.ProjectDir,
		TrackID:               tr.ID,
		CustomSkills:          customSkills,
		Mode:                  "full",
		ConfigMode:            cfgMode,
		SkipConfig:            cfgMode == installer.ConfigModeSkip,
		Backup:                doBackup,
		Force:                 true,
		Version:               m.cfg.Version,
		ProjectConfigRenderer: m.cfg.ProjectConfigRenderer,
	})
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	m.installRes = res
	m.botFrame = 1
}

func (m Model) updateDone(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.doneIdx > 0 {
			m.doneIdx--
		}
	case "down", "j":
		if m.doneIdx < 2 {
			m.doneIdx++
		}
	case "enter":
		switch m.doneIdx {
		case 0:
			m.prevScreen = screenDone
			m.guideTab = 0
			m.screen = screenGuide
		case 1:
			m.prevScreen = screenDone
			m.guideTab = 2
			m.skillCursor = 0
			m.screen = screenGuide
		case 2:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) updateGuide(key string) (tea.Model, tea.Cmd) {
	skills := m.visibleSkills()
	switch key {
	case "esc", "backspace":
		if m.cfg.StartInGuide {
			return m, tea.Quit
		}
		m.screen = m.prevScreen
	case "tab", "right", "l":
		m.guideTab = (m.guideTab + 1) % 3
	case "left", "h":
		m.guideTab = (m.guideTab + 2) % 3
	case "1":
		m.guideTab = 0
	case "2":
		m.guideTab = 1
	case "3":
		m.guideTab = 2
	case "a":
		if m.guideTab == 2 {
			m.showAllInGuide = !m.showAllInGuide
			m.skillCursor = 0
		}
	case "up", "k":
		if m.guideTab == 2 && m.skillCursor > 0 {
			m.skillCursor--
		}
	case "down", "j":
		if m.guideTab == 2 && m.skillCursor < len(skills)-1 {
			m.skillCursor++
		}
	}
	return m, nil
}

func (m *Model) selectTargetReceipt() state.Receipt {
	if len(m.receipts) > 0 {
		idx := m.updateIdx
		if idx < 0 || idx >= len(m.receipts) {
			idx = 0
		}
		return m.receipts[idx]
	}
	store := state.NewStore(m.cfg.Env.StateFilePath())
	paths, _ := m.cfg.Env.ResolvePaths(m.selectedHarness(), m.scope, m.cfg.ProjectDir)
	if found, ok, err := store.Find(m.selectedHarness(), m.scope, paths.RootDir); err == nil && ok && found != nil {
		return *found
	}
	if m.scope == "global" {
		receipts, _ := store.Load()
		for _, r := range receipts {
			if strings.EqualFold(r.Harness, m.selectedHarness()) && strings.EqualFold(r.Scope, "global") {
				return r
			}
		}
	}
	return state.Receipt{
		Harness: m.selectedHarness(),
		Scope:   m.scope,
		RootDir: paths.RootDir,
		Track:   m.selectedTrack().ID,
	}
}

func (m *Model) loadUpdateReport() {
	m.errMsg = ""
	m.updateReport = nil
	targetReceipt := m.selectTargetReceipt()
	rep, err := updater.Check(context.Background(), m.cfg.UpdateRunner, targetReceipt, m.cfg.Version)
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	m.updateReport = rep
}

func (m Model) updateUpdateScreen(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "backspace":
		m.screen = screenHome
	case "up", "k":
		if len(m.receipts) > 1 && m.updateIdx > 0 {
			m.updateIdx--
			m.statusMsg = ""
			m.loadUpdateReport()
		}
	case "down", "j":
		if len(m.receipts) > 1 && m.updateIdx < len(m.receipts)-1 {
			m.updateIdx++
			m.statusMsg = ""
			m.loadUpdateReport()
		}
	case "enter":
		if m.updateReport != nil && m.updateReport.UpdateAvailable {
			r := m.selectTargetReceipt()
			_, sha, err := updater.Apply(context.Background(), m.cfg.UpdateRunner, m.cfg.Env, r, updater.ApplyOptions{
				Backup:                true,
				Force:                 true,
				CurrentVersion:        m.cfg.Version,
				ProjectConfigRenderer: m.cfg.ProjectConfigRenderer,
			})
			if err != nil {
				m.errMsg = err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("Updated %s (%s) skills to latest commit %s.", HarnessDisplayName(r.Harness), r.Scope, shortSHA(sha))
				m.updateReport.UpdateAvailable = false
				m.loadReceipts()
			}
		}
	}
	return m, nil
}

func (m *Model) loadSnapshots() {
	mgr := backup.NewManager(m.cfg.Env.BackupsDir())
	snaps, err := mgr.ListSnapshots("")
	if err != nil {
		m.errMsg = err.Error()
		return
	}
	m.snapshots = snaps
	m.restoreIdx = 0
	m.confirmRestore = false
}

func (m Model) updateRestoreScreen(key string) (tea.Model, tea.Cmd) {
	if m.confirmRestore {
		switch key {
		case "esc", "backspace", "n", "N":
			m.confirmRestore = false
		case "enter", "y", "Y":
			m.confirmRestore = false
			if m.restoreIdx >= 0 && m.restoreIdx < len(m.snapshots) {
				mgr := backup.NewManager(m.cfg.Env.BackupsDir())
				snap := m.snapshots[m.restoreIdx]
				res, err := mgr.RestoreSnapshot(snap.ID, m.cfg.Env.StateFilePath())
				if err != nil {
					m.errMsg = err.Error()
				} else {
					m.statusMsg = fmt.Sprintf("Restored snapshot %s (%d files restored, %d created files removed).",
						snap.ID, len(res.RestoredFiles), len(res.RemovedFiles))
				}
			}
		}
		return m, nil
	}
	switch key {
	case "esc", "backspace":
		m.screen = screenHome
	case "up", "k":
		if m.restoreIdx > 0 {
			m.restoreIdx--
		}
	case "down", "j":
		if m.restoreIdx < len(m.snapshots)-1 {
			m.restoreIdx++
		}
	case "enter":
		if m.restoreIdx >= 0 && m.restoreIdx < len(m.snapshots) {
			m.confirmRestore = true
		}
	}
	return m, nil
}

func (m *Model) loadReceipts() {
	store := state.NewStore(m.cfg.Env.StateFilePath())
	list, _ := store.Load()
	if len(list) == 0 && m.cfg.Catalog != nil {
		for _, h := range harnessOrder {
			if leg, ok := state.DiscoverLegacyReceipt(m.cfg.Env, m.cfg.Catalog, h, "global", m.cfg.ProjectDir); ok {
				list = append(list, *leg)
			}
			if leg, ok := state.DiscoverLegacyReceipt(m.cfg.Env, m.cfg.Catalog, h, "project", m.cfg.ProjectDir); ok {
				list = append(list, *leg)
			}
		}
	}
	m.receipts = list
	m.uninstallIdx = 0
	m.confirmUninstall = false
}

func (m Model) updateUninstallScreen(key string) (tea.Model, tea.Cmd) {
	if m.confirmUninstall {
		switch key {
		case "esc", "backspace", "n", "N":
			m.confirmUninstall = false
		case "enter", "y", "Y":
			m.confirmUninstall = false
			if m.uninstallIdx >= 0 && m.uninstallIdx < len(m.receipts) {
				rec := m.receipts[m.uninstallIdx]
				res, err := state.Uninstall(m.cfg.Env, m.cfg.Catalog, rec, backup.NewManager(m.cfg.Env.BackupsDir()), true)
				if err != nil {
					m.errMsg = err.Error()
				} else {
					store := state.NewStore(m.cfg.Env.StateFilePath())
					_ = store.Remove(rec.Harness, rec.Scope, rec.RootDir)
					m.statusMsg = fmt.Sprintf("Uninstalled %s (%s): removed %d tracked files.",
						HarnessDisplayName(rec.Harness), rec.Scope, len(res.RemovedFiles))
					m.loadReceipts()
				}
			}
		}
		return m, nil
	}
	switch key {
	case "esc", "backspace":
		m.screen = screenHome
	case "up", "k":
		if m.uninstallIdx > 0 {
			m.uninstallIdx--
		}
	case "down", "j":
		if m.uninstallIdx < len(m.receipts)-1 {
			m.uninstallIdx++
		}
	case "enter":
		if m.uninstallIdx >= 0 && m.uninstallIdx < len(m.receipts) {
			m.confirmUninstall = true
		}
	}
	return m, nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func (m Model) View() string {
	var body string
	switch m.screen {
	case screenHome:
		body = m.viewHome()
	case screenHarness:
		body = m.viewHarness()
	case screenTrack:
		body = m.viewTrack()
	case screenSkillTable:
		body = m.viewSkillTable()
	case screenBackup:
		body = m.viewBackup()
	case screenDone:
		body = m.viewDone()
	case screenGuide:
		body = m.viewGuide()
	case screenUpdate:
		body = m.viewUpdate()
	case screenRestore:
		body = m.viewRestore()
	case screenUninstall:
		body = m.viewUninstall()
	}
	return PageStyle.Render(body)
}

func (m Model) cardStyle(selected bool) lipgloss.Style {
	if selected {
		return SelectedCardStyle.Width(CardWidth(m.width))
	}
	return UnselectedCardStyle.Width(CardWidth(m.width))
}

func (m Model) backupCardStyle(selected bool, isRecommended bool) lipgloss.Style {
	if !selected {
		return UnselectedCardStyle.Width(CardWidth(m.width))
	}
	if isRecommended {
		return SelectedBackupCardStyle.Width(CardWidth(m.width))
	}
	return SelectedWarnCardStyle.Width(CardWidth(m.width))
}

func (m Model) activeNavMode() string {
	switch m.screen {
	case screenGuide:
		return "guide"
	case screenUpdate:
		return "update"
	case screenRestore:
		return "restore"
	case screenUninstall:
		return "uninstall"
	default:
		return "install"
	}
}

func (m Model) renderTopNav() string {
	left := StepEyebrowStyle.Render("skill-installer v4.0 · SkillBot")
	active := m.activeNavMode()
	modes := []string{"install", "guide", "update", "restore", "uninstall"}
	var rendered []string
	for _, mode := range modes {
		if mode == active {
			rendered = append(rendered, StepPillStyle.Render(mode))
		} else {
			rendered = append(rendered, StepEyebrowStyle.Render(mode))
		}
	}
	right := strings.Join(rendered, "   ")
	totalW := CardWidth(m.width) + 2
	gap := totalW - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 3 {
		gap = 3
	}
	div := DividerStyle.Render(strings.Repeat("─", totalW))
	return left + strings.Repeat(" ", gap) + right + "\n" + div + "\n\n"
}

func (m Model) renderSubtext(text string) string {
	innerW := CardWidth(m.width) - 6
	return lipgloss.NewStyle().PaddingLeft(2).Width(innerW).Render(SubtitleStyle.Render(text))
}

func (m Model) renderBotHeader(stepLabel, speech string) string {
	var out strings.Builder
	out.WriteString(m.renderTopNav())

	lines := strings.Split(strings.TrimSpace(speech), "\n")
	headline := ""
	var rest []string
	if len(lines) > 0 {
		headline = lines[0]
		if len(lines) > 1 {
			rest = lines[1:]
		}
	}

	rightW := CardWidth(m.width) - 8
	var right strings.Builder
	if stepLabel != "" {
		right.WriteString(StepEyebrowStyle.Render(stepLabel) + "\n")
	}
	if headline != "" {
		right.WriteString(TitleStyle.Copy().Width(rightW).Render(headline))
	}
	if len(rest) > 0 {
		right.WriteString("\n" + SubtitleStyle.Copy().Width(rightW).Render(strings.Join(rest, "\n")))
	}

	joined := lipgloss.JoinHorizontal(lipgloss.Top, BotAvatar(m.botFrame), "   ", right.String())
	out.WriteString(joined + "\n\n")
	return out.String()
}

func (m Model) renderWizardFooter(backLabel, actionLabel, keyHint string) string {
	totalW := CardWidth(m.width) + 2
	div := DividerStyle.Render(strings.Repeat("─", totalW)) + "\n"

	left := StepEyebrowStyle.Render(backLabel)
	appName := HarnessDisplayName(m.selectedHarness())
	tr := m.selectedTrack()
	mid := StepEyebrowStyle.Render(fmt.Sprintf("%s · %s", appName, tr.Name))
	right := PrimaryButtonStyle.Render(actionLabel)

	used := lipgloss.Width(left) + lipgloss.Width(mid) + lipgloss.Width(right)
	rem := totalW - used
	gap1 := 3
	gap2 := 3
	if rem > 6 {
		gap1 = rem / 2
		gap2 = rem - gap1
	}
	row := left + strings.Repeat(" ", gap1) + mid + strings.Repeat(" ", gap2) + right + "\n"
	return "\n" + div + row + FooterStyle.Render(keyHint)
}

func (m Model) viewHome() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("WELCOME",
		"What would you like to do today?\nSet up curated AI workflows in 3 quick steps — existing config files are backed up or merged safely."))

	items := []struct {
		title string
		sub   string
	}{
		{"Choose a workflow track (60-sec guided setup)", "Safe by default: backs up, merges, or preserves your existing CLAUDE.md / GEMINI.md"},
		{"See how this works & command guide", "Plain-English tour of /slash commands, skills, tracks, and starter prompts"},
		{"Check for skill & workflow updates", "See what changed on main and update your installed track with automatic backup"},
		{"Restore a safety backup", "Roll back to a timestamped snapshot if you want a do-over"},
		{"Uninstall", "Cleanly remove plugin files while keeping your custom skills untouched"},
	}

	for idx, it := range items {
		selected := idx == m.homeIdx
		if selected {
			card := fmt.Sprintf("%s%s\n%s", StepPillStyle.Render("› "), TitleStyle.Render(it.title), m.renderSubtext(it.sub))
			b.WriteString(m.cardStyle(true).Render(card) + "\n")
		} else {
			card := fmt.Sprintf("  %s", TitleStyle.Render(it.title))
			b.WriteString(m.cardStyle(false).Render(card) + "\n")
		}
	}

	b.WriteString(m.renderWizardFooter("q: Quit", "Continue →", "Up/Down: move  ·  Enter: select  ·  ?: how this works  ·  q: quit"))
	return b.String()
}

func (m Model) viewHarness() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("STEP 1 OF 3 · TARGET APP & SCOPE",
		"Which AI app are you setting up?"))

	for idx, h := range harnessOrder {
		selected := idx == m.harnessIdx
		if selected {
			paths, _ := m.cfg.Env.ResolvePaths(h, m.scope, m.cfg.ProjectDir)
			pathHint := fmt.Sprintf("Skills: %s   Config: %s", paths.SkillsDir, paths.ConfigFile)
			innerW := CardWidth(m.width) - 6
			pathLine := lipgloss.NewStyle().PaddingLeft(2).Width(innerW).Render(StepPillStyle.Render(pathHint))
			card := fmt.Sprintf("%s%s\n%s\n%s",
				StepPillStyle.Render("› "),
				TitleStyle.Render(HarnessDisplayName(h)),
				m.renderSubtext(harnessTaglines[h]),
				pathLine,
			)
			b.WriteString(m.cardStyle(true).Render(card) + "\n")
		} else {
			card := fmt.Sprintf("  %s", TitleStyle.Render(HarnessDisplayName(h)))
			b.WriteString(m.cardStyle(false).Render(card) + "\n")
		}
	}

	scopeGlobal := "[ ] Global (all projects)"
	scopeProj := fmt.Sprintf("[ ] Specific project folder (%s)", m.cfg.ProjectDir)
	if m.scope == "global" {
		scopeGlobal = SlashStyle.Render("[x] Global (all projects)")
	} else {
		scopeProj = SlashStyle.Render(fmt.Sprintf("[x] Specific project folder (%s)", m.cfg.ProjectDir))
	}
	b.WriteString("\n" + StepEyebrowStyle.Render("Scope (Tab / e):") + "  " + scopeGlobal + "   " + scopeProj + "\n")

	if m.editingProjectDir {
		editBox := fmt.Sprintf("Target project folder: %s█\n  Press Enter to confirm folder (supports ~/...), Ctrl+U to clear, or Esc to cancel.",
			SlashStyle.Render(m.projectDirInput))
		if m.projectDirErr != "" {
			editBox += "\n  " + BadgeStyle.Render(m.projectDirErr)
		}
		b.WriteString("\n" + m.cardStyle(true).Render(editBox) + "\n")
		b.WriteString(FooterStyle.Render("Type folder path  ·  Enter: confirm folder  ·  Ctrl+U: clear  ·  Esc: cancel"))
		return b.String()
	}

	b.WriteString(m.renderWizardFooter("← Back", "Continue →", "Up/Down: choose app  ·  Tab: switch Global/Project  ·  e: set project folder  ·  Enter: next"))
	return b.String()
}

func (m Model) viewTrack() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("STEP 2 OF 3 · WORKFLOW TRACK",
		"How much structure do you want?"))

	innerW := CardWidth(m.width) - 6
	if m.cfg.Catalog != nil {
		for idx, tr := range m.cfg.Catalog.Tracks {
			selected := idx == m.trackIdx
			skills, _ := m.cfg.Catalog.TrackSkills(tr.ID, m.selectedHarness())
			countLabel := fmt.Sprintf("%d curated skills", len(skills))
			if tr.ID == "custom" {
				countLabel = fmt.Sprintf("%d selected", m.selectedCustomCount())
			}

			bestCue := ""
			if tr.BestWhen != "" {
				bestCue = TruncatePlain("Best when: "+tr.BestWhen, 42)
			}

			if selected {
				leftTitle := StepPillStyle.Render("› ") + TitleStyle.Render(tr.Name)
				rightTag := StepEyebrowStyle.Render(bestCue)
				gap := innerW - lipgloss.Width(leftTitle) - lipgloss.Width(rightTag)
				if gap < 2 {
					gap = 2
				}
				line1 := leftTitle + strings.Repeat(" ", gap) + rightTag
				desc := tr.Description
				if desc == "" {
					desc = tr.Tagline
				}
				fullDesc := fmt.Sprintf("(%s) %s", countLabel, desc)
				card := fmt.Sprintf("%s\n%s", line1, m.renderSubtext(fullDesc))
				b.WriteString(m.cardStyle(true).Render(card) + "\n")
			} else {
				leftTitle := "  " + TitleStyle.Render(tr.Name)
				rightTag := StepEyebrowStyle.Render(bestCue)
				gap := innerW - lipgloss.Width(leftTitle) - lipgloss.Width(rightTag)
				if gap < 2 {
					gap = 2
				}
				line1 := leftTitle + strings.Repeat(" ", gap) + rightTag
				b.WriteString(m.cardStyle(false).Render(line1) + "\n")
			}
		}
	}

	curTrack := m.selectedTrack()
	b.WriteString("\n" + PeekLinkStyle.Render(fmt.Sprintf("Peek inside %q (press p or Space for short & long descriptions)", curTrack.Name)) + "\n")

	b.WriteString(m.renderWizardFooter("← Back", "Continue →", "Up/Down: choose  ·  Space/p: peek skills & descriptions  ·  Enter: continue  ·  Esc: back"))
	return b.String()
}

func (m Model) viewSkillTable() string {
	var b strings.Builder
	tr := m.selectedTrack()
	skills := m.visibleSkills()

	stepLabel := "STEP 2 OF 3 · SKILL PREVIEW"
	speech := fmt.Sprintf("Included skills in %s (%s)\nPress 'd' or Enter on any row for its full explanation and starter prompt.",
		tr.Name, HarnessDisplayName(m.selectedHarness()))
	if tr.ID == "custom" {
		stepLabel = "STEP 2 OF 3 · CUSTOM SKILL PICKER"
		speech = "Pick individual skills for your custom setup\nPress Space to toggle [x], 'd' to expand long description, and Enter when ready."
	}
	b.WriteString(m.renderBotHeader(stepLabel, speech))

	start := 0
	pageSize := 8
	if m.skillCursor >= pageSize {
		start = m.skillCursor - pageSize + 1
	}
	end := start + pageSize
	if end > len(skills) {
		end = len(skills)
	}

	totalW := CardWidth(m.width) + 2
	b.WriteString(StepEyebrowStyle.Render(fmt.Sprintf("%s %s %s %s %s",
		PadRight("", 6),
		PadRight("SKILL", 22),
		PadRight("COMMAND", 19),
		PadRight("GROUP", 13),
		"WHAT IT DOES",
	)) + "\n")
	b.WriteString(DividerStyle.Render(strings.Repeat("─", totalW)) + "\n")

	for idx := start; idx < end; idx++ {
		sm := skills[idx]
		cursor := "  "
		if idx == m.skillCursor {
			cursor = StepPillStyle.Render("› ")
		}
		check := "   "
		if tr.ID == "custom" {
			if m.customSelected[sm.Name] {
				check = SlashStyle.Render("[x]")
			} else {
				check = "[ ]"
			}
		}
		slash := sm.SlashCommand
		if slash == "" {
			slash = "(auto)"
		}
		row := fmt.Sprintf("%s%s %s %s %s %s\n",
			cursor,
			PadRight(check, 4),
			PadRight(TitleStyle.Render(TruncatePlain(sm.Name, 21)), 22),
			PadRight(StepPillStyle.Render(TruncatePlain(slash, 18)), 19),
			PadRight(CategoryBadge(sm.Category), 13),
			SubtitleStyle.Render(TruncatePlain(sm.Short, 28)),
		)
		b.WriteString(row)

		if m.skillExpanded[sm.Name] {
			detail := fmt.Sprintf("Detail: %s\nTry typing: %q", sm.Long, sm.StarterPrompt)
			drawer := PromptBoxStyle.Copy().MarginLeft(4).Width(CardWidth(m.width) - 6).Render(detail)
			b.WriteString(drawer + "\n\n")
		}
	}

	b.WriteString(fmt.Sprintf("\n%s\n", StepEyebrowStyle.Render(fmt.Sprintf("Showing %d-%d of %d skills", start+1, end, len(skills)))))
	if tr.ID == "custom" {
		b.WriteString(m.renderWizardFooter("← Close table (Esc)", "Confirm Selection →", "Up/Down: move  ·  Space: toggle [x]  ·  d: expand description  ·  Enter: confirm  ·  Esc: back"))
	} else {
		b.WriteString(m.renderWizardFooter("← Close table (Esc)", "Continue →", "Up/Down: move  ·  Enter/d: expand long description  ·  c: continue to Step 3  ·  Esc: back"))
	}
	return b.String()
}

func (m Model) viewBackup() string {
	var b strings.Builder
	paths, _ := m.cfg.Env.ResolvePaths(m.selectedHarness(), m.scope, m.cfg.ProjectDir)
	cfgName := "config file"
	if paths.ConfigFile != "" {
		cfgName = filepath.Base(paths.ConfigFile)
	}
	hasExistingCfg := m.existingConfigPresent()

	speech := "Save a safety backup before changing files?"
	if hasExistingCfg {
		speech = fmt.Sprintf("I detected an existing %s at %s.\nHow should we update your %s and safety backup?",
			cfgName, paths.ConfigFile, cfgName)
	}
	b.WriteString(m.renderBotHeader("STEP 3 OF 3 · CONFIG & SAFETY BACKUP", speech))

	type backupOpt struct {
		title       string
		sub         string
		recommended bool
	}
	var opts []backupOpt
	if hasExistingCfg {
		opts = []backupOpt{
			{
				fmt.Sprintf("Add / update workflow section in existing %s + save backup (Recommended)", cfgName),
				fmt.Sprintf("Keeps your current rules in %s and appends/updates a managed workflow section after saving a backup in %s", cfgName, m.cfg.Env.BackupsDir()),
				true,
			},
			{
				fmt.Sprintf("Replace %s with clean track template + save backup", cfgName),
				fmt.Sprintf("Snapshots your current %s into %s, then writes the clean track template", cfgName, m.cfg.Env.BackupsDir()),
				true,
			},
			{
				fmt.Sprintf("Keep my existing %s untouched (install skills & commands only)", cfgName),
				fmt.Sprintf("Installs skills, subagents, and commands without modifying your current %s", cfgName),
				true,
			},
			{
				fmt.Sprintf("Replace %s without saving a backup", cfgName),
				fmt.Sprintf("Replaces %s and existing plugin files directly without recording a snapshot", cfgName),
				false,
			},
		}
	} else {
		opts = []backupOpt{
			{
				"Yes, save a backup first (Recommended)",
				fmt.Sprintf("Lets you restore previous files anytime on Windows, Linux, or macOS (%s)", m.cfg.Env.BackupsDir()),
				true,
			},
			{
				"No, skip backup (--no-backup)",
				"Writes your chosen track directly without saving a snapshot",
				false,
			},
		}
	}

	for idx, o := range opts {
		selected := idx == m.backupIdx
		if selected {
			titleRendered := TitleStyle.Render(o.title)
			cursor := StepPillStyle.Render("› ")
			if o.recommended {
				titleRendered = SlashStyle.Render(o.title)
				cursor = SlashStyle.Render("› ")
			}
			card := fmt.Sprintf("%s%s\n%s", cursor, titleRendered, m.renderSubtext(o.sub))
			b.WriteString(m.backupCardStyle(true, o.recommended).Render(card) + "\n")
		} else {
			card := fmt.Sprintf("  %s", TitleStyle.Render(o.title))
			b.WriteString(m.backupCardStyle(false, o.recommended).Render(card) + "\n")
		}
	}

	b.WriteString(m.renderWizardFooter("← Back", "Install Now", "Up/Down: choose  ·  Enter: install now  ·  Esc: back"))
	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder
	if m.errMsg != "" {
		b.WriteString(m.renderBotHeader("INSTALL STOPPED", fmt.Sprintf("Something went wrong: %s", m.errMsg)))
		b.WriteString(FooterStyle.Render("Press q to exit"))
		return b.String()
	}

	res := m.installRes
	appName := HarnessDisplayName(m.selectedHarness())
	tr := m.selectedTrack()

	b.WriteString(m.renderBotHeader("SETUP COMPLETE",
		fmt.Sprintf("%s is ready with %s.\nCopy one of the starter prompts below into your next session, or open the interactive tour.", appName, tr.Name)))

	if res != nil {
		cfgSummary := res.Paths.ConfigFile
		if res.ConfigAction != "" {
			cfgSummary = fmt.Sprintf("%s (%s)", res.Paths.ConfigFile, res.ConfigAction)
		}
		summary := fmt.Sprintf("App & Scope:    %s (%s)\nTrack:          %s (%d skills installed)\nSkills folder:  %s\nConfig file:    %s",
			appName, res.Scope, tr.Name, len(res.Skills), res.Paths.SkillsDir, cfgSummary)
		if res.BackupID != "" {
			summary += fmt.Sprintf("\nSafety backup:  %s (restore anytime with `skill-installer restore`)", res.BackupID)
		}
		starters := m.StarterPrompts(2)
		if len(starters) > 0 {
			summary += "\n\nTry typing right now in your next session:"
			for _, sm := range starters {
				slash := sm.SlashCommand
				if slash == "" {
					slash = sm.Name
				}
				summary += fmt.Sprintf("\n  • %s — %q", StepPillStyle.Render(slash), sm.StarterPrompt)
			}
		}
		b.WriteString(m.cardStyle(false).Render(summary) + "\n\n")
	}

	choices := []struct {
		title string
		sub   string
	}{
		{"See what you can type now (60-sec interactive tour)", "Plain-English guide to your new /slash commands, skills, and starter prompts"},
		{"Explore every installed skill & command in detail", "Browse short & long descriptions and copy-ready prompts for each skill"},
		{"All set — exit to terminal", "Jump straight into your AI coding or product session"},
	}

	for idx, ch := range choices {
		selected := idx == m.doneIdx
		if selected {
			b.WriteString(m.cardStyle(true).Render(fmt.Sprintf("%s%s\n%s", StepPillStyle.Render("› "), TitleStyle.Render(ch.title), m.renderSubtext(ch.sub))) + "\n")
		} else {
			b.WriteString(m.cardStyle(false).Render(fmt.Sprintf("  %s", TitleStyle.Render(ch.title))) + "\n")
		}
	}

	b.WriteString(m.renderWizardFooter("q: Quit", "Open Selected →", "Up/Down: choose  ·  Enter: select  ·  q: quit"))
	return b.String()
}

func (m Model) viewGuide() string {
	var b strings.Builder
	h := m.selectedHarness()
	appName := HarnessDisplayName(h)
	tr := m.selectedTrack()

	b.WriteString(m.renderBotHeader("INTERACTIVE GUIDE & COMMAND EXPLORER",
		fmt.Sprintf("Learn how %s loads your workflow, or browse every skill & starter prompt.", appName)))

	tabs := []string{
		"1. What You Can Type Now",
		"2. How This Toolkit Works",
		"3. Skill & Command Explorer",
	}
	var renderedTabs []string
	for idx, t := range tabs {
		if idx == m.guideTab {
			renderedTabs = append(renderedTabs, SelectedCardStyle.Copy().Padding(0, 1).Render(StepPillStyle.Render(t)))
		} else {
			renderedTabs = append(renderedTabs, UnselectedCardStyle.Copy().Padding(0, 1).Render(StepEyebrowStyle.Render(t)))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...) + "\n\n")

	switch m.guideTab {
	case 0:
		loadLines := HarnessLoadLines(h)
		var howBuilder strings.Builder
		howBuilder.WriteString(TitleStyle.Render(fmt.Sprintf("Slash Commands vs. Skills in %s", appName)) + "\n")
		howBuilder.WriteString(SubtitleStyle.Render("Type a /slash command to run a workflow explicitly, or describe your task in plain English so matching skills load automatically.") + "\n")
		for _, l := range loadLines {
			howBuilder.WriteString("  • " + SubtitleStyle.Render(l) + "\n")
		}
		b.WriteString(m.cardStyle(false).Render(strings.TrimRight(howBuilder.String(), "\n")) + "\n\n")

		b.WriteString(TitleStyle.Render("Try typing one of these in your next session:") + "\n\n")
		for idx, sm := range m.installedOrSelectedSkills() {
			if idx >= 4 {
				break
			}
			slash := sm.SlashCommand
			if slash == "" {
				slash = sm.Name
			}
			b.WriteString(fmt.Sprintf("  • %s — %s\n    %s\n\n",
				StepPillStyle.Render(slash),
				TitleStyle.Render(sm.Short),
				SubtitleStyle.Render(fmt.Sprintf("Prompt: %q", sm.StarterPrompt)),
			))
		}
	case 1:
		overview := ""
		if m.cfg.Catalog != nil {
			overview = strings.TrimSpace(m.cfg.Catalog.Explainers.RepoOverview)
		}
		b.WriteString(m.cardStyle(true).Render(TitleStyle.Render("Why Curated Tracks?")+"\n"+SubtitleStyle.Render(overview)) + "\n")
		if m.cfg.Catalog != nil {
			for _, c := range m.cfg.Catalog.Explainers.Concepts {
				card := fmt.Sprintf("%s\n%s", StepPillStyle.Render(c.Title), SubtitleStyle.Render(c.Summary))
				b.WriteString(m.cardStyle(false).Render(card) + "\n")
			}
		}
	case 2:
		skills := m.visibleSkills()
		totalForHarness := len(skills)
		if m.cfg.Catalog != nil {
			totalForHarness = len(m.cfg.Catalog.SkillsForHarness(h))
		}
		if m.showAllInGuide {
			b.WriteString(fmt.Sprintf("%s\n\n",
				StepPillStyle.Render(fmt.Sprintf("Showing: All %d skills (%s) — press 'a' to filter to %s", totalForHarness, appName, tr.Name))))
		} else {
			b.WriteString(fmt.Sprintf("%s\n\n",
				StepPillStyle.Render(fmt.Sprintf("Showing: %s (%d skills) — press 'a' for all %d skills (%s)", tr.Name, len(skills), totalForHarness, appName))))
		}

		start := 0
		pageSize := 6
		if m.skillCursor >= pageSize {
			start = m.skillCursor - pageSize + 1
		}
		end := start + pageSize
		if end > len(skills) {
			end = len(skills)
		}
		for idx := start; idx < end; idx++ {
			sm := skills[idx]
			cursor := "  "
			if idx == m.skillCursor {
				cursor = StepPillStyle.Render("› ")
			}
			slash := sm.SlashCommand
			if slash == "" {
				slash = "(auto)"
			}
			b.WriteString(fmt.Sprintf("%s%s %s %s %s\n",
				cursor,
				PadRight(TitleStyle.Render(TruncatePlain(sm.Name, 21)), 22),
				PadRight(StepPillStyle.Render(TruncatePlain(slash, 18)), 19),
				PadRight(CategoryBadge(sm.Category), 13),
				SubtitleStyle.Render(TruncatePlain(sm.Short, 31)),
			))
		}
		if m.skillCursor >= 0 && m.skillCursor < len(skills) {
			sel := skills[m.skillCursor]
			slash := sel.SlashCommand
			if slash == "" {
				slash = "(loaded automatically by your AI)"
			}
			innerW := CardWidth(m.width) - 6
			longWrapped := SubtitleStyle.Copy().Width(innerW).Render(sel.Long)
			card := fmt.Sprintf("%s  %s   Command: %s\n\n%s\n\nStarter prompt to try:\n  %s",
				TitleStyle.Render(sel.Name),
				CategoryBadge(sel.Category),
				StepPillStyle.Render(slash),
				longWrapped,
				SlashStyle.Render(sel.StarterPrompt),
			)
			b.WriteString("\n" + m.cardStyle(true).Render(card) + "\n")
		}
	}

	b.WriteString(m.renderWizardFooter("← Esc: Back", "Tab / 1-3: Switch Tab", "Tab / 1-3: switch section  ·  Up/Down: browse skills  ·  a: toggle track/all  ·  q: quit"))
	return b.String()
}

func (m Model) viewUpdate() string {
	var b strings.Builder
	targetReceipt := m.selectTargetReceipt()
	b.WriteString(m.renderBotHeader("SKILL & WORKFLOW UPDATES",
		fmt.Sprintf("Checking futuregerald/futuregerald-claude-plugin on main for %s (%s · %s)...",
			HarnessDisplayName(targetReceipt.Harness), targetReceipt.Scope, targetReceipt.RootDir)))

	if len(m.receipts) > 1 {
		for idx, r := range m.receipts {
			cursor := "  "
			selected := idx == m.updateIdx
			if selected {
				cursor = StepPillStyle.Render("› ")
			}
			line := fmt.Sprintf("%s%s (%s) — track: %s · %s",
				cursor, TitleStyle.Render(HarnessDisplayName(r.Harness)), r.Scope, r.Track, r.RootDir)
			b.WriteString(m.cardStyle(selected).Render(line) + "\n")
		}
	}

	if m.errMsg != "" {
		b.WriteString(m.cardStyle(false).Render("Couldn't reach GitHub via gh CLI (run 'gh auth login', or type /plugin-update in chat): "+m.errMsg) + "\n")
	} else if m.statusMsg != "" {
		b.WriteString(m.cardStyle(true).Render(SlashStyle.Render(m.statusMsg)) + "\n")
	} else if m.updateReport != nil {
		rep := m.updateReport
		if !rep.UpdateAvailable {
			b.WriteString(m.cardStyle(true).Render(fmt.Sprintf("You're completely up to date on main (%s).", shortSHA(rep.LatestSHA))) + "\n")
		} else {
			changedLabel := strings.Join(rep.ChangedSkills, ", ")
			if changedLabel == "" {
				changedLabel = "config, templates, or agent files"
			}
			info := fmt.Sprintf("Update available on main: %s -> %s\nChanged skills: %s\nPress Enter to back up & apply update now",
				shortSHA(rep.InstalledSHA), shortSHA(rep.LatestSHA), changedLabel)
			b.WriteString(m.cardStyle(true).Render(info) + "\n")
		}
	}
	footer := "Enter: apply update  ·  Esc: back to home  ·  q: quit"
	if len(m.receipts) > 1 {
		footer = "Up/Down: select target  ·  " + footer
	}
	b.WriteString(m.renderWizardFooter("← Back to Setup", "Apply Update →", footer))
	return b.String()
}

func (m Model) viewRestore() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("RESTORE A SAFETY BACKUP",
		"Pick a timestamped snapshot below to restore overwritten files and remove files created in that run."))

	if m.statusMsg != "" {
		b.WriteString(m.cardStyle(true).Render(SlashStyle.Render(m.statusMsg)) + "\n")
	}
	if m.confirmRestore && m.restoreIdx >= 0 && m.restoreIdx < len(m.snapshots) {
		snap := m.snapshots[m.restoreIdx]
		b.WriteString(m.cardStyle(true).Render(BadgeStyle.Render(
			fmt.Sprintf("Confirm restore of snapshot %s? Press Enter/y to confirm, Esc/n to cancel.", snap.ID))) + "\n")
	}
	if len(m.snapshots) == 0 {
		b.WriteString(m.cardStyle(false).Render("No safety backup snapshots found yet. Press Esc and select 'Choose a workflow track (60-sec guided setup)' to install your first track.") + "\n")
	} else {
		for idx, s := range m.snapshots {
			cursor := "  "
			selected := idx == m.restoreIdx
			if selected {
				cursor = StepPillStyle.Render("› ")
			}
			line := fmt.Sprintf("%s%s   (%s · %s · track: %s · %d tracked entries)",
				cursor, TitleStyle.Render(s.ID), HarnessDisplayName(s.Harness), s.Scope, s.Track, len(s.Entries))
			b.WriteString(m.cardStyle(selected).Render(line) + "\n")
		}
	}
	b.WriteString(m.renderWizardFooter("← Back to Setup", "Restore Snapshot →", "Up/Down: select snapshot  ·  Enter: restore  ·  Esc: back  ·  q: quit"))
	return b.String()
}

func (m Model) viewUninstall() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("SAFE UNINSTALL",
		"Select an installation below to remove tracked plugin files while keeping your custom skills and *.local.md files untouched."))

	if m.statusMsg != "" {
		b.WriteString(m.cardStyle(true).Render(SlashStyle.Render(m.statusMsg)) + "\n")
	}
	if m.confirmUninstall && m.uninstallIdx >= 0 && m.uninstallIdx < len(m.receipts) {
		rec := m.receipts[m.uninstallIdx]
		b.WriteString(m.cardStyle(true).Render(BadgeStyle.Render(
			fmt.Sprintf("Confirm uninstall of %s (%s)? Press Enter/y to confirm, Esc/n to cancel.",
				HarnessDisplayName(rec.Harness), rec.Scope))) + "\n")
	}
	if len(m.receipts) == 0 {
		b.WriteString(m.cardStyle(false).Render("No tracked installations found. Press Esc and select 'Choose a workflow track (60-sec guided setup)' to install your first track.") + "\n")
	} else {
		for idx, r := range m.receipts {
			cursor := "  "
			selected := idx == m.uninstallIdx
			if selected {
				cursor = StepPillStyle.Render("› ")
			}
			line := fmt.Sprintf("%s%s (%s) — track: %s · %d files at %s",
				cursor, TitleStyle.Render(HarnessDisplayName(r.Harness)), r.Scope, r.Track, len(r.InstalledFiles), r.RootDir)
			b.WriteString(m.cardStyle(selected).Render(line) + "\n")
		}
	}
	b.WriteString(m.renderWizardFooter("← Back to Setup", "Uninstall →", "Up/Down: select install  ·  Enter: uninstall  ·  Esc: back  ·  q: quit"))
	return b.String()
}
