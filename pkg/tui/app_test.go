package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/dotsaver/pkg/manifest"
	"github.com/user/dotsaver/pkg/syncer"
)

func TestTUIModelRender(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	view := m.View()
	if !strings.Contains(view, "dotsaver") {
		t.Errorf("Expected 'dotsaver' title in view output, got: %s", view)
	}
}

func TestTUIModelInit(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Expected non-nil cmd from Init()")
	}
}

func TestTUIModelTabSwitching(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	if m.activeTab != TabDashboard {
		t.Fatalf("Expected initial tab TabDashboard, got %v", m.activeTab)
	}

	// Switch to TabPrograms via '2'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = res.(*Model)
	if m.activeTab != TabPrograms {
		t.Errorf("Expected activeTab TabPrograms, got %v", m.activeTab)
	}
	view := m.View()
	if !strings.Contains(view, "Programs") {
		t.Errorf("Expected view to show Programs, got: %s", view)
	}

	// Switch to TabSettings via '3'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m = res.(*Model)
	if m.activeTab != TabSettings {
		t.Errorf("Expected activeTab TabSettings, got %v", m.activeTab)
	}
	view = m.View()
	if !strings.Contains(view, "Configuration") {
		t.Errorf("Expected view to show Configuration, got: %s", view)
	}

	// Switch back via Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = res.(*Model)
	if m.activeTab != TabDashboard {
		t.Errorf("Expected activeTab TabDashboard after Esc in settings, got %v", m.activeTab)
	}

	// Switch using Tab key on Dashboard
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = res.(*Model)
	if m.activeTab != TabPrograms {
		t.Errorf("Expected activeTab TabPrograms after Tab key, got %v", m.activeTab)
	}
}

func TestTUIWindowSize(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	res, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = res.(*Model)
	if m.width != 120 || m.height != 40 {
		t.Errorf("Expected dimensions 120x40, got %dx%d", m.width, m.height)
	}
}

func TestTUIProgramsAddAndDelete(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}
	m.cfg.RepoPath = filepath.Join(tempDir, "repo")

	// Switch to TabPrograms
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = res.(*Model)

	// Press 'a' to open Add Program dialog
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = res.(*Model)
	if !m.programsView.AddingProgram {
		t.Fatal("Expected AddingProgram to be true")
	}

	// Type "tmux" into ProgNameInput
	m.programsView.ProgNameInput.SetValue("tmux")

	// Submit via Enter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*Model)

	if m.programsView.AddingProgram {
		t.Error("Expected AddingProgram to be false after Enter")
	}
	if len(m.manifest.Programs) != 1 || m.manifest.Programs[0].Name != "tmux" {
		t.Fatalf("Expected 1 program 'tmux', got: %+v", m.manifest.Programs)
	}

	// Press 't' to add target
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	m = res.(*Model)
	if !m.programsView.AddingTarget {
		t.Fatal("Expected AddingTarget to be true")
	}

	m.programsView.TargetOSInput.SetValue("linux")
	m.programsView.TargetPathInput.SetValue("~/.tmux.conf")
	m.programsView.TargetAliasInput.SetValue(".tmux.conf")

	// Submit via Enter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(*Model)

	if m.programsView.AddingTarget {
		t.Error("Expected AddingTarget to be false after Enter")
	}
	prog := m.manifest.FindProgram("tmux")
	if prog == nil || len(prog.Targets) != 1 {
		t.Fatalf("Expected 1 target for 'tmux', got: %+v", prog)
	}

	// Delete target: focus on targets pane
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = res.(*Model)
	if m.programsView.FocusPane != 1 {
		t.Errorf("Expected FocusPane 1 (targets), got %d", m.programsView.FocusPane)
	}

	// Delete target with 'd'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = res.(*Model)
	prog = m.manifest.FindProgram("tmux")
	if len(prog.Targets) != 0 {
		t.Errorf("Expected 0 targets after deletion, got %d", len(prog.Targets))
	}

	// Switch back to Programs pane and delete program
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = res.(*Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = res.(*Model)
	if len(m.manifest.Programs) != 0 {
		t.Errorf("Expected 0 programs after deletion, got %d", len(m.manifest.Programs))
	}
}

func TestTUIModals(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	// 1. Conflict Modal
	m.activeModal = ModalConflict
	m.conflicts = []syncer.ConflictItem{
		{
			Program:    "git",
			TargetPath: "/path/to/.gitconfig",
			RepoPath:   "/repo/data/git/linux/.gitconfig",
		},
	}

	view := m.View()
	if !strings.Contains(view, "Sync Conflicts Detected") {
		t.Errorf("Expected conflict modal header in view, got: %s", view)
	}
	if !strings.Contains(view, "/path/to/.gitconfig") {
		t.Errorf("Expected conflict file path in view, got: %s", view)
	}

	// Esc triggers sync without backup (skips conflicts) and enters ModalBusy
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = res.(*Model)
	if m.activeModal != ModalBusy {
		t.Errorf("Expected activeModal ModalBusy after Esc, got %v", m.activeModal)
	}
	if cmd == nil {
		t.Fatal("Expected cmd from triggerSyncWithoutBackup")
	}

	// 2. Busy Modal
	m.spinnerText = "Synchronizing files..."

	busyView := m.View()
	if !strings.Contains(busyView, "Please Wait") {
		t.Errorf("Expected 'Please Wait' in busy view, got: %s", busyView)
	}
	if !strings.Contains(busyView, "Synchronizing files...") {
		t.Errorf("Expected spinnerText in busy view, got: %s", busyView)
	}
}

func TestTUIDashboardAddKey(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	// Press 'a' on Dashboard -> switches to TabPrograms and starts AddProgram
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = res.(*Model)
	if m.activeTab != TabPrograms {
		t.Errorf("Expected activeTab TabPrograms, got %v", m.activeTab)
	}
	if !m.programsView.AddingProgram {
		t.Error("Expected AddingProgram to be true")
	}
}

func TestTUIAsyncMessages(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	// gitStatusMsg
	res, _ := m.Update(gitStatusMsg{status: "Clean", isClean: true, branch: "main"})
	m = res.(*Model)
	if m.gitStatus != "Clean" || !m.isGitClean || m.gitBranch != "main" {
		t.Errorf("Git status not updated properly: %+v", m)
	}

	// saveDoneMsg
	res, _ = m.Update(saveDoneMsg{
		report: &syncer.SaveReport{
			Saved:    []string{"nvim -> ~/.config/nvim"},
			Warnings: []string{},
		},
		pushOut: "pushed",
		err:     nil,
	})
	m = res.(*Model)
	if !strings.Contains(m.statusMsg, "Saved 1 target(s) successfully") {
		t.Errorf("Unexpected statusMsg: %s", m.statusMsg)
	}

	// syncDoneMsg
	res, _ = m.Update(syncDoneMsg{
		report: &syncer.SyncReport{
			Updated:  []string{"~/.config/nvim"},
			BackedUp: []string{"~/.config/nvim.bak"},
		},
		manifest: manifest.New(),
		err:      nil,
	})
	m = res.(*Model)
	if !strings.Contains(m.statusMsg, "Sync complete: 1 restored, 1 backed up") {
		t.Errorf("Unexpected statusMsg: %s", m.statusMsg)
	}

	// saveSettingsDoneMsg
	res, _ = m.Update(saveSettingsDoneMsg{err: nil})
	m = res.(*Model)
	if !strings.Contains(m.statusMsg, "Settings saved") {
		t.Errorf("Unexpected statusMsg: %s", m.statusMsg)
	}
}

func TestTUIQuitCommands(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	m, err := NewModel(cfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize model: %v", err)
	}

	// 'q' on Dashboard
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("Expected Quit cmd on 'q'")
	}

	// 'ctrl+c' anywhere
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Expected Quit cmd on 'ctrl+c'")
	}
}
