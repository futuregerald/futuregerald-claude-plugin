package tui

import (
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
	if !strings.Contains(homeView, "SkillBot") || !strings.Contains(homeView, "Set up skills & workflow") {
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

	sendKey("p")
	if m.screen != screenSkillTable {
		t.Fatalf("expected screenSkillTable on 'p', got %v", m.screen)
	}
	sendKey("d")
	tableView := m.View()
	if !strings.Contains(tableView, "Try typing:") {
		t.Errorf("expected expanded skill long description and starter prompt in table view:\n%s", tableView)
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
	if !strings.Contains(doneView, "See what you can type now") || m.installRes == nil || m.installRes.BackupID == "" {
		t.Errorf("expected completion view with tour CTA and backup ID, got:\n%s", doneView)
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
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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
