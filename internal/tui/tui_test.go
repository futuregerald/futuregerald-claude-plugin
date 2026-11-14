package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/state"
)

func TestGuidedWizardInstallAndPostInstallTour(t *testing.T) {
	cat, err := catalog.Load(catalog.FS)
	if err != nil {
		t.Fatalf("catalog.Load error: %v", err)
	}

	tmpHome := t.TempDir()
	projDir := t.TempDir()
	env := platform.Env{
		GOOS:    "darwin",
		GOARCH:  "arm64",
		HomeDir: tmpHome,
	}

	m := NewModel(Config{
		Env:           env,
		Catalog:       cat,
		InitialScope:  "project",
		InitialTrack:  "basics",
		InitialBackup: true,
		ProjectDir:    projDir,
		Version:       "4.0.0",
	})

	sendKey := func(k string) {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		m = updated.(Model)
	}
	sendSpecial := func(kt tea.KeyType) {
		updated, _ := m.Update(tea.KeyMsg{Type: kt})
		m = updated.(Model)
	}

	homeView := m.View()
	if !strings.Contains(homeView, "SkillBot") || !strings.Contains(homeView, "Choose a workflow track") {
		t.Fatalf("unexpected home view:\n%s", homeView)
	}

	sendSpecial(tea.KeyEnter)
	if m.screen != screenHarness {
		t.Fatalf("expected screenHarness, got %v", m.screen)
	}

	sendSpecial(tea.KeyDown)
	sendSpecial(tea.KeyEnter)
	if m.screen != screenTrack {
		t.Fatalf("expected screenTrack, got %v", m.screen)
	}
	trackView := m.View()
	if !strings.Contains(trackView, "Best when:") {
		t.Errorf("expected 'Best when:' cue on track cards:\n%s", trackView)
	}

	sendKey("p")
	if m.screen != screenSkillTable {
		t.Fatalf("expected screenSkillTable on 'p', got %v", m.screen)
	}
	sendKey("d")
	tableView := m.View()
	if !strings.Contains(tableView, "Try typing:") || !strings.Contains(tableView, "STEP 2 OF 3") {
		t.Errorf("expected expanded skill long description, starter prompt, and STEP 2 OF 3 pill in table view:\n%s", tableView)
	}
	sendSpecial(tea.KeyEsc)
	if m.screen != screenTrack {
		t.Fatalf("expected back to screenTrack, got %v", m.screen)
	}

	sendSpecial(tea.KeyEnter)
	if m.screen != screenBackup {
		t.Fatalf("expected screenBackup, got %v", m.screen)
	}

	sendSpecial(tea.KeyEnter)
	if m.screen != screenDone {
		t.Fatalf("expected screenDone after install, got %v (err=%s)", m.screen, m.errMsg)
	}
	doneView := m.View()
	if !strings.Contains(doneView, "SETUP COMPLETE") || !strings.Contains(doneView, "See what you can type now") || !strings.Contains(doneView, "Try typing right now") || m.installRes == nil || m.installRes.BackupID == "" {
		t.Errorf("expected completion view with SETUP COMPLETE, starter prompts, tour CTA, and backup ID, got:\n%s", doneView)
	}

	sendSpecial(tea.KeyEnter)
	if m.screen != screenGuide {
		t.Fatalf("expected screenGuide from post-install tour CTA, got %v", m.screen)
	}
	guideView := m.View()
	if !strings.Contains(guideView, "What You Can Type Now") {
		t.Errorf("unexpected guide view:\n%s", guideView)
	}
	sendKey("3")
	explorerView := m.View()
	if !strings.Contains(explorerView, "Starter prompt to try:") {
		t.Errorf("expected Skill & Command Explorer detail card:\n%s", explorerView)
	}
}

func TestHeadlessTeaProgramAndPlainRenderers(t *testing.T) {
	cat, err := catalog.Load(catalog.FS)
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(Config{
		Env:          platform.Env{GOOS: "linux", GOARCH: "amd64", HomeDir: t.TempDir()},
		Catalog:      cat,
		StartInGuide: true,
		Version:      "4.0.0",
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = updated.(Model)
	if m.guideTab != 1 {
		t.Fatalf("expected guideTab 1, got %d", m.guideTab)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatalf("expected quit cmd on 'q'")
	}

	plainGuide := RenderPlainGuide(cat, "zed", "product")
	if !strings.Contains(plainGuide, "Zed Editor") || !strings.Contains(plainGuide, "Product & Strategy Flow") {
		t.Errorf("unexpected RenderPlainGuide output:\n%s", plainGuide)
	}
}

func TestRestoreUninstallConfirmAndCustomDeselect(t *testing.T) {
	cat, err := catalog.Load(catalog.FS)
	if err != nil {
		t.Fatal(err)
	}
	tmpHome := t.TempDir()
	projDir := t.TempDir()
	env := platform.Env{
		GOOS:    "darwin",
		GOARCH:  "arm64",
		HomeDir: tmpHome,
	}

	m := NewModel(Config{
		Env:        env,
		Catalog:    cat,
		ProjectDir: projDir,
		Version:    "4.0.0",
	})
	if m.selectedCustomCount() != 1 || !m.customSelected["plugin-updater"] {
		t.Fatalf("expected custom picker to start with only plugin-updater preselected, got %d (%v)", m.selectedCustomCount(), m.customSelected)
	}
	m.customSelected["baton-pass"] = true
	m.harnessIdx = 1
	if m.selectedCustomCount() != 1 {
		t.Fatalf("expected Antigravity-only skill excluded from Claude custom count, got %d", m.selectedCustomCount())
	}

	updated, tickCmd := m.Update(tickMsg{})
	m = updated.(Model)
	if tickCmd == nil {
		t.Fatalf("expected tickMsg to schedule next animation frame")
	}

	plain := RenderPlainGuide(cat, "claude", "basics")
	for _, nonASCII := range []string{"─", "•", "—"} {
		if strings.Contains(plain, nonASCII) {
			t.Errorf("expected pure-ASCII RenderPlainGuide without %q", nonASCII)
		}
	}

	m.customSelected = map[string]bool{}
	m.customSelected["brainstorming"] = true
	m.customSelected["writing-plans"] = true
	if m.selectedCustomCount() != 2 {
		t.Fatalf("expected 2 custom skills selected, got %d", m.selectedCustomCount())
	}
	m.customSelected["writing-plans"] = false
	if m.selectedCustomCount() != 1 {
		t.Fatalf("expected 1 custom skill selected after setting false, got %d", m.selectedCustomCount())
	}

	m.screen = screenUninstall
	m.receipts = []state.Receipt{{Harness: "claude", Scope: "project", RootDir: projDir, Track: "basics"}}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !m.confirmUninstall {
		t.Fatalf("expected first Enter on screenUninstall to prompt confirmUninstall")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = updated.(Model)
	if m.confirmUninstall {
		t.Fatalf("expected 'n' to cancel confirmUninstall")
	}
}

func TestStep1CustomProjectDirInlineEditAndStep3MergeReplaceSkip(t *testing.T) {
	cat, err := catalog.Load(catalog.FS)
	if err != nil {
		t.Fatal(err)
	}
	tmpHome := t.TempDir()
	defaultProj := t.TempDir()
	customProj := t.TempDir()
	env := platform.Env{
		GOOS:    "darwin",
		GOARCH:  "arm64",
		HomeDir: tmpHome,
	}

	m := NewModel(Config{
		Env:           env,
		Catalog:       cat,
		InitialScope:  "global",
		InitialTrack:  "basics",
		InitialBackup: true,
		ProjectDir:    defaultProj,
		Version:       "4.0.0",
	})

	sendKey := func(k string) {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		m = updated.(Model)
	}
	sendSpecial := func(kt tea.KeyType) {
		updated, _ := m.Update(tea.KeyMsg{Type: kt})
		m = updated.(Model)
	}

	sendSpecial(tea.KeyEnter)
	if m.screen != screenHarness {
		t.Fatalf("expected screenHarness, got %v", m.screen)
	}
	sendSpecial(tea.KeyDown)

	sendKey("e")
	if !m.editingProjectDir {
		t.Fatalf("expected 'e' on screenHarness to start editing project directory")
	}
	sendSpecial(tea.KeyCtrlU)
	sendKey(customProj)
	if m.screen != screenHarness {
		t.Fatalf("expected typing path with 'q' or '?' not to leave screenHarness")
	}
	sendSpecial(tea.KeyEnter)
	if m.editingProjectDir {
		t.Fatalf("expected Enter to commit custom project directory (err=%s)", m.projectDirErr)
	}
	if m.scope != "project" || m.cfg.ProjectDir != customProj {
		t.Fatalf("expected scope=project and ProjectDir=%q, got scope=%q ProjectDir=%q", customProj, m.scope, m.cfg.ProjectDir)
	}

	existingClaudeMD := customProj + "/CLAUDE.md"
	if err := writeTestFile(existingClaudeMD, "# Existing Custom Rules\n\nDo not lose this line.\n"); err != nil {
		t.Fatal(err)
	}

	sendSpecial(tea.KeyEnter)
	sendSpecial(tea.KeyEnter)
	if m.screen != screenBackup {
		t.Fatalf("expected screenBackup, got %v", m.screen)
	}
	backupView := m.View()
	if !strings.Contains(backupView, "Add / update workflow section") || !strings.Contains(backupView, "Replace CLAUDE.md with clean track template") || !strings.Contains(backupView, "Keep my existing CLAUDE.md untouched") {
		t.Fatalf("expected merge, replace, and keep-untouched options in Step 3 view:\n%s", backupView)
	}

	sendSpecial(tea.KeyEnter)
	if m.screen != screenDone || m.installRes == nil {
		t.Fatalf("expected screenDone after merge install, got %v (err=%s)", m.screen, m.errMsg)
	}
	if m.installRes.ConfigAction != "merged" {
		t.Errorf("expected ConfigAction=merged, got %q", m.installRes.ConfigAction)
	}
	mergedContent, err := readTestFile(existingClaudeMD)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mergedContent, "Do not lose this line.") || !strings.Contains(mergedContent, "BEGIN SKILL-INSTALLER WORKFLOW RULES") {
		t.Errorf("expected existing custom rules + managed workflow section in %s, got:\n%s", existingClaudeMD, mergedContent)
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func readTestFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}
