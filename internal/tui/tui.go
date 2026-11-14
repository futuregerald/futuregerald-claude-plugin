package tui

import (
	"context"
	"fmt"
	"os"
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
	cfg              Config
	screen           screenID
	prevScreen       screenID
	width            int
	height           int
	botFrame         int
	homeIdx          int
	harnessIdx       int
	scope            string
	trackIdx         int
	backupIdx        int
	doneIdx          int
	guideTab         int
	skillCursor      int
	skillExpanded    map[string]bool
	customSelected   map[string]bool
	showAllInGuide   bool
	installRes       *installer.InstallResult
	statusMsg        string
	errMsg           string
	updateReport     *updater.CheckReport
	snapshots        []backup.Manifest
	restoreIdx       int
	confirmRestore   bool
	receipts         []state.Receipt
	uninstallIdx     int
	confirmUninstall bool
}

var harnessOrder = []string{
	catalog.HarnessAntigravity,
	catalog.HarnessClaude,
	catalog.HarnessZed,
	catalog.HarnessOpenCode,
}

var harnessTaglines = map[string]string{
	catalog.HarnessAntigravity: "Installs GEMINI.md, native /slash skills, subagents, and optional circuit-breaker hooks",
	catalog.HarnessClaude:      "Installs CLAUDE.md, .claude/skills, subagents, and /start, /shape, /write-a-prd slash commands",
	catalog.HarnessZed:         "Installs AGENTS.md and .agents/skills (automatically exposed as /slash commands in Zed)",
	catalog.HarnessOpenCode:    "Installs AGENTS.md, .opencode/skills, subagents, and .opencode/commands",
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

	customSel := make(map[string]bool)
	if cfg.Catalog != nil {
		if defSkills, err := cfg.Catalog.TrackSkills("opinionated", harnessOrder[hIdx]); err == nil {
			for _, s := range defSkills {
				customSel[s] = true
			}
		}
		customSel["plugin-updater"] = true
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

	if m.screen == screenGuide && m.showAllInGuide {
		return m.cfg.Catalog.SkillsForHarness(h)
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

func (m Model) selectedCustomCount() int {
	count := 0
	for _, v := range m.customSelected {
		if v {
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
	info, err := os.Stat(paths.ConfigFile)
	return err == nil && !info.IsDir()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		m.botFrame = (m.botFrame + 1) % len(botFrames)
		return m, nil
	case tea.KeyMsg:
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
	case "enter":
		m.screen = screenTrack
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
			m.screen = screenBackup
			return m, nil
		}
		if m.skillCursor >= 0 && m.skillCursor < len(skills) {
			name := skills[m.skillCursor].Name
			m.skillExpanded[name] = !m.skillExpanded[name]
		}
	case "c":
		m.screen = screenBackup
	}
	return m, nil
}

func (m Model) updateBackup(key string) (tea.Model, tea.Cmd) {
	maxIdx := 1
	if m.existingConfigPresent() {
		maxIdx = 2
	}
	switch key {
	case "esc", "backspace":
		m.screen = screenTrack
	case "up", "k", "left":
		if m.backupIdx > 0 {
			m.backupIdx--
		}
	case "down", "j", "right":
		if m.backupIdx < maxIdx {
			m.backupIdx++
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

	skipConfig := m.backupIdx == 1 && m.existingConfigPresent()
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
		SkipConfig:            skipConfig,
		Backup:                m.backupIdx == 0,
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
	case "enter", " ":
		if m.guideTab == 2 && m.skillCursor >= 0 && m.skillCursor < len(skills) {
			name := skills[m.skillCursor].Name
			m.skillExpanded[name] = !m.skillExpanded[name]
		}
	}
	return m, nil
}

func (m *Model) selectTargetReceipt() state.Receipt {
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
				m.statusMsg = fmt.Sprintf("Updated skills to latest commit %s!", shortSHA(sha))
				m.updateReport.UpdateAvailable = false
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

func (m Model) renderBotHeader(stepLabel, speech string) string {
	line1 := fmt.Sprintf("%s  %s", BotAvatar(m.botFrame), TitleStyle.Render("SkillBot · skill-installer v4.0"))
	if stepLabel != "" {
		line1 += "   " + StepPillStyle.Render(stepLabel)
	}
	content := line1 + "\n\n" + SubtitleStyle.Render(speech)
	return BotBoxStyle.Render(content) + "\n\n"
}

func (m Model) viewHome() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("WELCOME",
		"\"Hey there! I set up curated AI workflows, slash commands, and skills without burying you in 60+ random files.\"\n\"Pick one thing below — we'll take it one step at a time.\""))

	items := []struct {
		title string
		sub   string
	}{
		{"Set up skills & workflow", "Guided 3-question setup for Antigravity, Claude Code, Zed, or OpenCode"},
		{"See how this works & command guide", "Plain-English tour of /slash commands, skills, tracks, and starter prompts"},
		{"Check for updates (gh)", "Pull the latest skills and improvements directly from main"},
		{"Restore a safety backup", "Roll back to a timestamped snapshot if you want a do-over"},
		{"Uninstall", "Cleanly remove plugin files while keeping your custom skills untouched"},
	}

	for idx, it := range items {
		cursor := "  "
		style := UnselectedCardStyle
		if idx == m.homeIdx {
			cursor = "> "
			style = SelectedCardStyle
		}
		card := fmt.Sprintf("%s%s\n  %s", cursor, TitleStyle.Render(it.title), SubtitleStyle.Render(it.sub))
		b.WriteString(style.Render(card) + "\n")
	}

	b.WriteString(FooterStyle.Render("Up/Down: move  ·  Enter: select  ·  ?: how this works  ·  q: quit"))
	return b.String()
}

func (m Model) viewHarness() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("STEP 1 OF 3 · TARGET APP",
		"\"Which AI coding or product assistant are we configuring today?\""))

	for idx, h := range harnessOrder {
		cursor := "  "
		style := UnselectedCardStyle
		if idx == m.harnessIdx {
			cursor = "> "
			style = SelectedCardStyle
		}
		paths, _ := m.cfg.Env.ResolvePaths(h, m.scope, m.cfg.ProjectDir)
		pathHint := fmt.Sprintf("Skills: %s   Config: %s", paths.SkillsDir, paths.ConfigFile)
		card := fmt.Sprintf("%s%s\n  %s\n  %s",
			cursor,
			TitleStyle.Render(HarnessDisplayName(h)),
			SubtitleStyle.Render(harnessTaglines[h]),
			StepPillStyle.Render(pathHint),
		)
		b.WriteString(style.Render(card) + "\n")
	}

	scopeGlobal := "[ ] Global (all projects)"
	scopeProj := "[ ] Current project folder"
	if m.scope == "global" {
		scopeGlobal = SlashStyle.Render("[x] Global (all projects)")
	} else {
		scopeProj = SlashStyle.Render("[x] Current project folder")
	}
	b.WriteString("\n  Scope (press Tab to switch):   " + scopeGlobal + "     " + scopeProj + "\n")
	b.WriteString(FooterStyle.Render("Up/Down: choose app  ·  Tab: switch Global/Project  ·  Enter: next  ·  ?: guide  ·  Esc: back"))
	return b.String()
}

func (m Model) viewTrack() string {
	var b strings.Builder
	appName := HarnessDisplayName(m.selectedHarness())
	b.WriteString(m.renderBotHeader("STEP 2 OF 3 · WORKFLOW TRACK",
		fmt.Sprintf("\"You don't need every skill installed at once. Pick the track that matches how you use %s.\"\n\"Press Space or 'p' on any track to peek inside its colorful skill table and long descriptions!\"", appName)))

	if m.cfg.Catalog != nil {
		for idx, tr := range m.cfg.Catalog.Tracks {
			cursor := "  "
			style := UnselectedCardStyle
			if idx == m.trackIdx {
				cursor = "> "
				style = SelectedCardStyle
			}
			skills, _ := m.cfg.Catalog.TrackSkills(tr.ID, m.selectedHarness())
			countLabel := fmt.Sprintf("%d curated skills", len(skills))
			if tr.ID == "custom" {
				countLabel = fmt.Sprintf("%d selected skills (press Space to customize)", m.selectedCustomCount())
			}
			header := fmt.Sprintf("%s%s   %s   %s",
				cursor,
				TitleStyle.Render(tr.Name),
				BadgeStyle.Render("["+tr.Badge+"]"),
				StepPillStyle.Render(countLabel),
			)
			card := fmt.Sprintf("%s\n  %s\n  %s",
				header,
				SlashStyle.Render(tr.Tagline),
				SubtitleStyle.Render(tr.Description),
			)
			b.WriteString(style.Render(card) + "\n")
		}
	}

	b.WriteString(FooterStyle.Render("Up/Down: choose track  ·  Space/p: peek skills & descriptions  ·  Enter: continue  ·  Esc: back"))
	return b.String()
}

func (m Model) viewSkillTable() string {
	var b strings.Builder
	tr := m.selectedTrack()
	skills := m.visibleSkills()

	speech := fmt.Sprintf("\"Here is what's inside %s for %s. Press 'd' or Enter on any skill to read its full description and starter prompt.\"",
		tr.Name, HarnessDisplayName(m.selectedHarness()))
	if tr.ID == "custom" {
		speech = "\"Toggle any skill with Space, press 'd' to expand its long description, and press Enter when your mix is ready.\""
	}
	b.WriteString(m.renderBotHeader("SKILL & COMMAND INSPECTOR", speech))

	start := 0
	pageSize := 10
	if m.skillCursor >= pageSize {
		start = m.skillCursor - pageSize + 1
	}
	end := start + pageSize
	if end > len(skills) {
		end = len(skills)
	}

	b.WriteString(fmt.Sprintf("  %s %s %s %s %s\n",
		PadRight("", 4),
		PadRight("SKILL", 28),
		PadRight("CATEGORY", 18),
		PadRight("SLASH COMMAND", 18),
		"SUMMARY",
	))
	b.WriteString("  " + strings.Repeat("─", 96) + "\n")

	for idx := start; idx < end; idx++ {
		sm := skills[idx]
		cursor := "  "
		if idx == m.skillCursor {
			cursor = "> "
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
			PadRight(TitleStyle.Render(sm.Name), 28),
			PadRight(CategoryBadge(sm.Category), 18),
			PadRight(SlashStyle.Render(slash), 18),
			sm.Short,
		)
		b.WriteString(row)

		if m.skillExpanded[sm.Name] {
			detail := fmt.Sprintf("Detail: %s\nTry typing: %q", sm.Long, sm.StarterPrompt)
			b.WriteString("      " + PromptBoxStyle.Render(detail) + "\n\n")
		}
	}

	b.WriteString(fmt.Sprintf("\n  Showing %d–%d of %d skills\n", start+1, end, len(skills)))
	if tr.ID == "custom" {
		b.WriteString(FooterStyle.Render("Up/Down: move  ·  Space: toggle [x]  ·  d: expand long description  ·  Enter: confirm selection  ·  Esc: back"))
	} else {
		b.WriteString(FooterStyle.Render("Up/Down: move  ·  Enter/d: expand long description  ·  c: continue to install  ·  Esc: back to tracks"))
	}
	return b.String()
}

func (m Model) viewBackup() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("STEP 3 OF 3 · SAFETY BACKUP",
		"\"Before I write any files, should I save a timestamped safety backup of anything I overwrite?\"\n\"It's on by default so you can always run `skill-installer restore` if you change your mind.\""))

	opts := []struct {
		title string
		sub   string
	}{
		{
			"Yes, save a safety backup first (Recommended)",
			fmt.Sprintf("Saves a clean snapshot in %s before touching any existing files", m.cfg.Env.BackupsDir()),
		},
		{
			"No, skip backup and keep existing config file untouched (--no-backup)",
			"Installs skills without a snapshot while preserving any existing CLAUDE.md / GEMINI.md / AGENTS.md",
		},
	}
	if m.existingConfigPresent() {
		opts = append(opts, struct {
			title string
			sub   string
		}{
			"No, skip backup and overwrite existing config file",
			"Overwrites your existing config file without saving a snapshot first",
		})
	}

	for idx, o := range opts {
		cursor := "  "
		style := UnselectedCardStyle
		if idx == m.backupIdx {
			cursor = "> "
			style = SelectedCardStyle
		}
		card := fmt.Sprintf("%s%s\n  %s", cursor, TitleStyle.Render(o.title), SubtitleStyle.Render(o.sub))
		b.WriteString(style.Render(card) + "\n")
	}

	b.WriteString(FooterStyle.Render("Up/Down: choose  ·  Enter: install now  ·  Esc: back"))
	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder
	if m.errMsg != "" {
		b.WriteString(m.renderBotHeader("OOPS", fmt.Sprintf("\"Something went wrong: %s\"", m.errMsg)))
		b.WriteString(FooterStyle.Render("Press q to exit"))
		return b.String()
	}

	res := m.installRes
	appName := HarnessDisplayName(m.selectedHarness())
	tr := m.selectedTrack()

	b.WriteString(m.renderBotHeader("ALL CONFIGURED!",
		fmt.Sprintf("\"High five! %s is ready with the %s track.\"\n\"Want a quick 60-second tour of what you can type in your very next chat session?\"", appName, tr.Name)))

	if res != nil {
		summary := fmt.Sprintf("App & Scope:    %s (%s)\nTrack:          %s (%d skills installed)\nSkills folder:  %s\nConfig file:    %s",
			appName, res.Scope, tr.Name, len(res.Skills), res.Paths.SkillsDir, res.Paths.ConfigFile)
		if res.BackupID != "" {
			summary += fmt.Sprintf("\nSafety backup:  %s (restore anytime with `skill-installer restore`)", res.BackupID)
		}
		b.WriteString(UnselectedCardStyle.Render(summary) + "\n\n")
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
		cursor := "  "
		style := UnselectedCardStyle
		if idx == m.doneIdx {
			cursor = "> "
			style = SelectedCardStyle
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%s\n  %s", cursor, TitleStyle.Render(ch.title), SubtitleStyle.Render(ch.sub))) + "\n")
	}

	b.WriteString(FooterStyle.Render("Up/Down: choose  ·  Enter: select  ·  q: quit"))
	return b.String()
}

func (m Model) viewGuide() string {
	var b strings.Builder
	appName := HarnessDisplayName(m.selectedHarness())
	tr := m.selectedTrack()

	b.WriteString(m.renderBotHeader("INTERACTIVE GUIDE & COMMAND EXPLORER",
		fmt.Sprintf("\"Here's everything you need to know about using %s with %s — zero jargon, promise!\"", tr.Name, appName)))

	tabs := []string{
		"1. What You Can Type Now",
		"2. How This Toolkit Works",
		"3. Skill & Command Explorer",
	}
	var renderedTabs []string
	for idx, t := range tabs {
		if idx == m.guideTab {
			renderedTabs = append(renderedTabs, SelectedCardStyle.Render(TitleStyle.Render(t)))
		} else {
			renderedTabs = append(renderedTabs, UnselectedCardStyle.Render(SubtitleStyle.Render(t)))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...) + "\n\n")

	switch m.guideTab {
	case 0:
		if m.cfg.Catalog != nil {
			for _, c := range m.cfg.Catalog.Explainers.Concepts {
				card := fmt.Sprintf("%s\n%s", BadgeStyle.Render(c.Title), c.Summary)
				b.WriteString(UnselectedCardStyle.Render(card) + "\n")
			}
		}
		b.WriteString(TitleStyle.Render("Try typing one of these in your next session:") + "\n\n")
		for idx, sm := range m.visibleSkills() {
			if idx >= 4 {
				break
			}
			b.WriteString(fmt.Sprintf("  • %s — %s\n    %s\n\n",
				SlashStyle.Render(sm.SlashCommand),
				sm.Short,
				SubtitleStyle.Render(fmt.Sprintf("Prompt: %q", sm.StarterPrompt)),
			))
		}
	case 1:
		overview := ""
		if m.cfg.Catalog != nil {
			overview = strings.TrimSpace(m.cfg.Catalog.Explainers.RepoOverview)
		}
		b.WriteString(UnselectedCardStyle.Render(TitleStyle.Render("Why Curated Tracks?")+"\n"+overview) + "\n")
		b.WriteString(UnselectedCardStyle.Render(TitleStyle.Render("Seamless Updates (gh CLI & GitHub MCP)")+
			"\n• Ask your AI directly: \"Are there updates to the plugin?\" or type /plugin-update.\n"+
			"• Or run `skill-installer update` from your terminal to pull the latest skills from main.") + "\n")
		b.WriteString(UnselectedCardStyle.Render(TitleStyle.Render("Safety Backups, Restore & Uninstall")+
			"\n• `skill-installer restore` rolls back any snapshot saved before an install or update.\n"+
			"• `skill-installer uninstall` removes only tracked plugin files, leaving your custom skills safe.") + "\n")
	case 2:
		skills := m.visibleSkills()
		start := 0
		pageSize := 8
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
				cursor = "> "
			}
			slash := sm.SlashCommand
			if slash == "" {
				slash = "(auto)"
			}
			b.WriteString(fmt.Sprintf("%s%s %s %s %s\n",
				cursor,
				PadRight(TitleStyle.Render(sm.Name), 28),
				PadRight(CategoryBadge(sm.Category), 18),
				PadRight(SlashStyle.Render(slash), 18),
				sm.Short,
			))
		}
		if m.skillCursor >= 0 && m.skillCursor < len(skills) {
			sel := skills[m.skillCursor]
			slash := sel.SlashCommand
			if slash == "" {
				slash = "(loaded automatically by your AI)"
			}
			card := fmt.Sprintf("%s  %s   Slash: %s\n\n%s\n\nStarter prompt to try:\n  %s",
				TitleStyle.Render(sel.Name),
				CategoryBadge(sel.Category),
				SlashStyle.Render(slash),
				sel.Long,
				SlashStyle.Render(sel.StarterPrompt),
			)
			b.WriteString("\n" + SelectedCardStyle.Render(card) + "\n")
		}
	}

	b.WriteString(FooterStyle.Render("Tab / 1-3: switch section  ·  Up/Down: browse skills  ·  a: toggle track/all skills  ·  Esc: back  ·  q: quit"))
	return b.String()
}

func (m Model) viewUpdate() string {
	var b strings.Builder
	targetReceipt := m.selectTargetReceipt()
	b.WriteString(m.renderBotHeader("LIVE UPDATER (GITHUB CLI)",
		fmt.Sprintf("\"Checking `futuregerald/futuregerald-claude-plugin` on `main` for %s (%s · %s)...\"",
			HarnessDisplayName(targetReceipt.Harness), targetReceipt.Scope, targetReceipt.RootDir)))

	if m.errMsg != "" {
		b.WriteString(UnselectedCardStyle.Render("Update check note: "+m.errMsg) + "\n")
	} else if m.statusMsg != "" {
		b.WriteString(SelectedCardStyle.Render(SlashStyle.Render(m.statusMsg)) + "\n")
	} else if m.updateReport != nil {
		rep := m.updateReport
		if !rep.UpdateAvailable {
			b.WriteString(SelectedCardStyle.Render(fmt.Sprintf("You're completely up to date on main (%s)!", shortSHA(rep.LatestSHA))) + "\n")
		} else {
			info := fmt.Sprintf("Update available on main: %s -> %s\nChanged skills: %v\nPress Enter to backup & apply update now!",
				shortSHA(rep.InstalledSHA), shortSHA(rep.LatestSHA), rep.ChangedSkills)
			b.WriteString(SelectedCardStyle.Render(info) + "\n")
		}
	}
	b.WriteString(FooterStyle.Render("Enter: apply update  ·  Esc: back to home  ·  q: quit"))
	return b.String()
}

func (m Model) viewRestore() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("RESTORE A SAFETY BACKUP",
		"\"Pick a timestamped snapshot below to restore overwritten files and remove files created in that run.\""))

	if m.statusMsg != "" {
		b.WriteString(SelectedCardStyle.Render(SlashStyle.Render(m.statusMsg)) + "\n")
	}
	if m.confirmRestore && m.restoreIdx >= 0 && m.restoreIdx < len(m.snapshots) {
		snap := m.snapshots[m.restoreIdx]
		b.WriteString(SelectedCardStyle.Render(BadgeStyle.Render(
			fmt.Sprintf("Confirm restore of snapshot %s? Press Enter/y to confirm, Esc/n to cancel.", snap.ID))) + "\n")
	}
	if len(m.snapshots) == 0 {
		b.WriteString(UnselectedCardStyle.Render("No safety backup snapshots found yet.") + "\n")
	} else {
		for idx, s := range m.snapshots {
			cursor := "  "
			style := UnselectedCardStyle
			if idx == m.restoreIdx {
				cursor = "> "
				style = SelectedCardStyle
			}
			line := fmt.Sprintf("%s%s   (%s · %s · track: %s · %d tracked entries)",
				cursor, TitleStyle.Render(s.ID), HarnessDisplayName(s.Harness), s.Scope, s.Track, len(s.Entries))
			b.WriteString(style.Render(line) + "\n")
		}
	}
	b.WriteString(FooterStyle.Render("Up/Down: select snapshot  ·  Enter: restore  ·  Esc: back  ·  q: quit"))
	return b.String()
}

func (m Model) viewUninstall() string {
	var b strings.Builder
	b.WriteString(m.renderBotHeader("SAFE UNINSTALL",
		"\"Select an installation below to remove tracked plugin files while keeping your custom skills and *.local.md files untouched.\""))

	if m.statusMsg != "" {
		b.WriteString(SelectedCardStyle.Render(SlashStyle.Render(m.statusMsg)) + "\n")
	}
	if m.confirmUninstall && m.uninstallIdx >= 0 && m.uninstallIdx < len(m.receipts) {
		rec := m.receipts[m.uninstallIdx]
		b.WriteString(SelectedCardStyle.Render(BadgeStyle.Render(
			fmt.Sprintf("Confirm uninstall of %s (%s)? Press Enter/y to confirm, Esc/n to cancel.",
				HarnessDisplayName(rec.Harness), rec.Scope))) + "\n")
	}
	if len(m.receipts) == 0 {
		b.WriteString(UnselectedCardStyle.Render("No tracked installations found.") + "\n")
	} else {
		for idx, r := range m.receipts {
			cursor := "  "
			style := UnselectedCardStyle
			if idx == m.uninstallIdx {
				cursor = "> "
				style = SelectedCardStyle
			}
			line := fmt.Sprintf("%s%s (%s) — track: %s · %d files at %s",
				cursor, TitleStyle.Render(HarnessDisplayName(r.Harness)), r.Scope, r.Track, len(r.InstalledFiles), r.RootDir)
			b.WriteString(style.Render(line) + "\n")
		}
	}
	b.WriteString(FooterStyle.Render("Up/Down: select install  ·  Enter: uninstall  ·  Esc: back  ·  q: quit"))
	return b.String()
}
