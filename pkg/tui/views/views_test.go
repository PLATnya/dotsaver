package views

import (
	"strings"
	"testing"

	"github.com/user/dotsaver/pkg/manifest"
	"github.com/user/dotsaver/pkg/syncer"
)

func TestRenderDashboard(t *testing.T) {
	data := DashboardData{
		RemoteURL:  "git@github.com:user/dotfiles.git",
		RepoPath:   "/path/to/repo",
		GitBranch:  "main",
		GitStatus:  "Clean",
		IsGitClean: true,
		CurrentOS:  "windows",
		Programs: []ProgramSummary{
			{
				Name:        "nvim",
				TargetCount: 2,
				Paths:       []string{"~/AppData/Local/nvim"},
			},
		},
		TotalPrograms: 1,
		TotalTargets:  1,
	}

	out := RenderDashboard(data, 80)
	if !strings.Contains(out, "Repository & Status") {
		t.Errorf("Expected 'Repository & Status' in dashboard output, got: %s", out)
	}
	if !strings.Contains(out, "git@github.com:user/dotfiles.git") {
		t.Errorf("Expected remote URL in dashboard output, got: %s", out)
	}
	if !strings.Contains(out, "nvim") {
		t.Errorf("Expected program 'nvim' in dashboard output, got: %s", out)
	}

	// Test with zero programs
	dataEmpty := DashboardData{
		CurrentOS: "linux",
		GitStatus: "Uninitialized",
	}
	outEmpty := RenderDashboard(dataEmpty, 80)
	if !strings.Contains(outEmpty, "No programs registered yet") {
		t.Errorf("Expected 'No programs registered yet', got: %s", outEmpty)
	}
}

func TestRenderPrograms(t *testing.T) {
	pv := NewProgramsView()
	m := manifest.New()
	_, _ = m.AddProgram("zsh")
	_ = m.AddTarget("zsh", manifest.Target{OS: "linux", Path: "~/.zshrc", Alias: ".zshrc"})

	out := pv.Render(m, "linux", 80)
	if !strings.Contains(out, "zsh") {
		t.Errorf("Expected 'zsh' in programs output, got: %s", out)
	}
	if !strings.Contains(out, "~/.zshrc") {
		t.Errorf("Expected '~/.zshrc' in target inspector, got: %s", out)
	}

	// Test Add Program Dialog Render
	pv.StartAddProgram()
	outAddProg := pv.Render(m, "linux", 80)
	if !strings.Contains(outAddProg, "Add New Program") {
		t.Errorf("Expected 'Add New Program' dialog title, got: %s", outAddProg)
	}

	// Test Add Target Dialog Render
	pv.CancelDialog()
	pv.StartAddTarget("linux", "zsh")
	outAddTarget := pv.Render(m, "linux", 80)
	if !strings.Contains(outAddTarget, "Add Target to 'zsh'") {
		t.Errorf("Expected 'Add Target to 'zsh'', got: %s", outAddTarget)
	}
}

func TestRenderSettings(t *testing.T) {
	sv := NewSettingsView("https://github.com/test/dots.git", "/cache/repo")

	// Default mode must be SettingsModeSelect
	if sv.Mode != SettingsModeSelect {
		t.Fatalf("Expected default mode SettingsModeSelect, got %v", sv.Mode)
	}
	if sv.RemoteURLInput.Focused() || sv.RepoPathInput.Focused() {
		t.Errorf("Expected inputs to be blurred in SettingsModeSelect")
	}

	out := sv.Render(80)
	if !strings.Contains(out, "dotsaver Configuration") {
		t.Errorf("Expected 'dotsaver Configuration', got: %s", out)
	}
	if !strings.Contains(out, "https://github.com/test/dots.git") {
		t.Errorf("Expected remote URL in settings output, got: %s", out)
	}
	if !strings.Contains(out, "SELECT MODE") {
		t.Errorf("Expected 'SELECT MODE' badge in output, got: %s", out)
	}
	if !strings.Contains(out, "[i]") {
		t.Errorf("Expected '[i]' key hint in output, got: %s", out)
	}

	// Switch to Input Mode
	sv.SetMode(SettingsModeInput)
	if sv.Mode != SettingsModeInput {
		t.Fatalf("Expected mode SettingsModeInput after SetMode, got %v", sv.Mode)
	}
	if !sv.RemoteURLInput.Focused() {
		t.Errorf("Expected RemoteURLInput to be focused in InputMode")
	}

	outInput := sv.Render(80)
	if !strings.Contains(outInput, "INPUT MODE") {
		t.Errorf("Expected 'INPUT MODE' badge in output, got: %s", outInput)
	}
	if !strings.Contains(outInput, "[Alt+i") {
		t.Errorf("Expected '[Alt+i' key hint in output, got: %s", outInput)
	}

	// Switch back to Select Mode
	sv.SetMode(SettingsModeSelect)
	if sv.Mode != SettingsModeSelect {
		t.Fatalf("Expected mode SettingsModeSelect after SetMode, got %v", sv.Mode)
	}
	if sv.RemoteURLInput.Focused() || sv.RepoPathInput.Focused() {
		t.Errorf("Expected inputs to be blurred in SettingsModeSelect")
	}
}

func TestRenderModals(t *testing.T) {
	conflicts := []syncer.ConflictItem{
		{
			Program:    "git",
			TargetPath: "~/.gitconfig",
			RepoPath:   "/cache/data/git/linux/.gitconfig",
		},
	}

	cOut := RenderConflictModal(conflicts, 80, 24)
	if !strings.Contains(cOut, "Sync Conflicts Detected") {
		t.Errorf("Expected 'Sync Conflicts Detected', got: %s", cOut)
	}
	if !strings.Contains(cOut, "~/.gitconfig") {
		t.Errorf("Expected conflict file '~/.gitconfig', got: %s", cOut)
	}

	bOut := RenderBusyModal("Cloning repository...", "⠋", 80, 24)
	if !strings.Contains(bOut, "Please Wait") {
		t.Errorf("Expected 'Please Wait', got: %s", bOut)
	}
	if !strings.Contains(bOut, "Cloning repository...") {
		t.Errorf("Expected spinnerText in modal, got: %s", bOut)
	}
}
