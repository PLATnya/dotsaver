package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/dotsaver/pkg/manifest"
)

func TestManifestAddAndFind(t *testing.T) {
	m := manifest.New()
	prog, err := m.AddProgram("nvim")
	if err != nil {
		t.Fatalf("AddProgram failed: %v", err)
	}
	if prog.Name != "nvim" {
		t.Errorf("Expected name nvim, got %s", prog.Name)
	}

	_, err = m.AddProgram("nvim")
	if err == nil {
		t.Errorf("Expected duplicate program error, got nil")
	}

	// Case-insensitive duplicate check
	_, err = m.AddProgram("NVIM")
	if err == nil {
		t.Errorf("Expected case-insensitive duplicate error, got nil")
	}

	// Empty name check
	_, err = m.AddProgram("   ")
	if err == nil {
		t.Errorf("Expected empty program name error, got nil")
	}

	found := m.FindProgram("nvim")
	if found == nil {
		t.Fatalf("FindProgram returned nil")
	}

	// Case-insensitive find
	foundUpper := m.FindProgram("NVIM")
	if foundUpper == nil {
		t.Fatalf("FindProgram with uppercase returned nil")
	}

	err = m.AddTarget("nvim", manifest.Target{OS: "windows", Path: "%LOCALAPPDATA%/nvim"})
	if err != nil {
		t.Fatalf("AddTarget failed: %v", err)
	}

	if len(found.Targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(found.Targets))
	}
	if found.Targets[0].Alias != "nvim" {
		t.Errorf("Expected default alias 'nvim', got %s", found.Targets[0].Alias)
	}
}

func TestManifestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	m := manifest.New()
	_, err := m.AddProgram("git")
	if err != nil {
		t.Fatalf("AddProgram failed: %v", err)
	}
	err = m.AddTarget("git", manifest.Target{OS: "linux", Path: "~/.gitconfig"})
	if err != nil {
		t.Fatalf("AddTarget failed: %v", err)
	}

	if err := m.Save(tempDir); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tempDir, manifest.ManifestFilename))
	if err != nil {
		t.Fatalf("Failed to read manifest file: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("Manifest file is empty")
	}

	loaded, err := manifest.Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if loaded.Version != manifest.CurrentVersion {
		t.Errorf("Expected version %d, got %d", manifest.CurrentVersion, loaded.Version)
	}

	if len(loaded.Programs) != 1 {
		t.Fatalf("Expected 1 program, got %d", len(loaded.Programs))
	}
	targets := loaded.TargetsForOS("linux")
	if len(targets["git"]) != 1 {
		t.Errorf("Expected 1 linux target for git, got %d", len(targets["git"]))
	}
	if targets["git"][0].Alias != ".gitconfig" {
		t.Errorf("Expected alias '.gitconfig', got %s", targets["git"][0].Alias)
	}
}

func TestManifestRemoveProgram(t *testing.T) {
	m := manifest.New()
	_, _ = m.AddProgram("tmux")
	_, _ = m.AddProgram("zsh")

	if !m.RemoveProgram("tmux") {
		t.Errorf("Expected RemoveProgram to return true for existing program")
	}

	if m.FindProgram("tmux") != nil {
		t.Errorf("Expected tmux to be removed")
	}

	if m.RemoveProgram("tmux") {
		t.Errorf("Expected RemoveProgram to return false for already removed program")
	}

	// Case-insensitive removal
	if !m.RemoveProgram("ZSH") {
		t.Errorf("Expected RemoveProgram to return true for case-insensitive match")
	}

	if len(m.Programs) != 0 {
		t.Errorf("Expected 0 programs remaining, got %d", len(m.Programs))
	}
}

func TestManifestAddTargetValidation(t *testing.T) {
	m := manifest.New()
	_, _ = m.AddProgram("alacritty")

	// Target for non-existent program
	err := m.AddTarget("nonexistent", manifest.Target{OS: "windows", Path: "alacritty.toml"})
	if err == nil {
		t.Errorf("Expected error for non-existent program, got nil")
	}

	// Invalid OS
	err = m.AddTarget("alacritty", manifest.Target{OS: "darwin", Path: "alacritty.toml"})
	if err == nil {
		t.Errorf("Expected error for unsupported OS, got nil")
	}

	// Empty path
	err = m.AddTarget("alacritty", manifest.Target{OS: "windows", Path: ""})
	if err == nil {
		t.Errorf("Expected error for empty path, got nil")
	}

	// Valid target with custom alias
	err = m.AddTarget("alacritty", manifest.Target{OS: "windows", Path: "%APPDATA%/alacritty/alacritty.toml", Alias: "config.toml"})
	if err != nil {
		t.Fatalf("AddTarget with custom alias failed: %v", err)
	}
	prog := m.FindProgram("alacritty")
	if prog.Targets[0].Alias != "config.toml" {
		t.Errorf("Expected alias 'config.toml', got %s", prog.Targets[0].Alias)
	}

	// Duplicate target for same OS and path
	err = m.AddTarget("alacritty", manifest.Target{OS: "windows", Path: "%APPDATA%/alacritty/alacritty.toml"})
	if err == nil {
		t.Errorf("Expected error for duplicate target, got nil")
	}

	// Same path for different OS is allowed
	err = m.AddTarget("alacritty", manifest.Target{OS: "linux", Path: "%APPDATA%/alacritty/alacritty.toml"})
	if err != nil {
		t.Errorf("Expected success for same path on different OS, got: %v", err)
	}
}

func TestManifestRemoveTarget(t *testing.T) {
	m := manifest.New()
	_, _ = m.AddProgram("vscode")
	_ = m.AddTarget("vscode", manifest.Target{OS: "windows", Path: "%APPDATA%/Code/User/settings.json"})
	_ = m.AddTarget("vscode", manifest.Target{OS: "linux", Path: "~/.config/Code/User/settings.json"})

	// Remove non-existent program target
	if m.RemoveTarget("nonexistent", "windows", "%APPDATA%/Code/User/settings.json") {
		t.Errorf("Expected false for non-existent program")
	}

	// Remove non-existent target path
	if m.RemoveTarget("vscode", "windows", "wrong/path") {
		t.Errorf("Expected false for non-existent target path")
	}

	// Remove existing target with case-insensitive OS
	if !m.RemoveTarget("vscode", "WINDOWS", "%APPDATA%/Code/User/settings.json") {
		t.Errorf("Expected true for successful target removal")
	}

	prog := m.FindProgram("vscode")
	if len(prog.Targets) != 1 {
		t.Fatalf("Expected 1 target remaining, got %d", len(prog.Targets))
	}
	if prog.Targets[0].OS != "linux" {
		t.Errorf("Expected remaining target to be linux, got %s", prog.Targets[0].OS)
	}
}

func TestManifestTargetsForOS(t *testing.T) {
	m := manifest.New()
	_, _ = m.AddProgram("fish")
	_ = m.AddTarget("fish", manifest.Target{OS: "linux", Path: "~/.config/fish/config.fish"})

	_, _ = m.AddProgram("pwsh")
	_ = m.AddTarget("pwsh", manifest.Target{OS: "windows", Path: "$PROFILE"})
	_ = m.AddTarget("pwsh", manifest.Target{OS: "linux", Path: "~/.config/powershell/Microsoft.PowerShell_profile.ps1"})

	linuxTargets := m.TargetsForOS("Linux")
	if len(linuxTargets) != 2 {
		t.Fatalf("Expected 2 programs for Linux, got %d", len(linuxTargets))
	}
	if len(linuxTargets["fish"]) != 1 {
		t.Errorf("Expected 1 target for fish, got %d", len(linuxTargets["fish"]))
	}
	if len(linuxTargets["pwsh"]) != 1 {
		t.Errorf("Expected 1 target for pwsh, got %d", len(linuxTargets["pwsh"]))
	}

	windowsTargets := m.TargetsForOS("windows")
	if len(windowsTargets) != 1 {
		t.Fatalf("Expected 1 program for Windows, got %d", len(windowsTargets))
	}
	if _, ok := windowsTargets["pwsh"]; !ok {
		t.Errorf("Expected pwsh in windows targets")
	}

	macosTargets := m.TargetsForOS("darwin")
	if len(macosTargets) != 0 {
		t.Errorf("Expected 0 programs for darwin, got %d", len(macosTargets))
	}
}

func TestManifestLoadNotExist(t *testing.T) {
	tempDir := t.TempDir()
	m, err := manifest.Load(tempDir)
	if err != nil {
		t.Fatalf("Expected no error loading non-existent manifest, got: %v", err)
	}
	if m.Version != manifest.CurrentVersion {
		t.Errorf("Expected version %d, got %d", manifest.CurrentVersion, m.Version)
	}
	if len(m.Programs) != 0 {
		t.Errorf("Expected 0 programs, got %d", len(m.Programs))
	}
}

func TestManifestLoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, manifest.ManifestFilename)
	if err := os.WriteFile(manifestFile, []byte("{ invalid json }"), 0644); err != nil {
		t.Fatalf("Failed to write invalid json: %v", err)
	}

	_, err := manifest.Load(tempDir)
	if err == nil {
		t.Errorf("Expected error loading invalid json, got nil")
	}
}

func TestManifestAliasFallbackToProgramName(t *testing.T) {
	m := manifest.New()
	_, _ = m.AddProgram("mytool")
	err := m.AddTarget("mytool", manifest.Target{OS: "linux", Path: "/"})
	if err != nil {
		t.Fatalf("AddTarget failed: %v", err)
	}
	prog := m.FindProgram("mytool")
	if prog.Targets[0].Alias != "mytool" {
		t.Errorf("Expected fallback alias 'mytool', got %s", prog.Targets[0].Alias)
	}

	err = m.AddTarget("mytool", manifest.Target{OS: "windows", Path: "."})
	if err != nil {
		t.Fatalf("AddTarget failed: %v", err)
	}
	if prog.Targets[1].Alias != "mytool" {
		t.Errorf("Expected fallback alias 'mytool', got %s", prog.Targets[1].Alias)
	}
}

func TestManifestLoadDefaultVersion(t *testing.T) {
	tempDir := t.TempDir()
	manifestFile := filepath.Join(tempDir, manifest.ManifestFilename)
	// JSON with no version field and null programs
	if err := os.WriteFile(manifestFile, []byte(`{"programs": null}`), 0644); err != nil {
		t.Fatalf("Failed to write json: %v", err)
	}

	m, err := manifest.Load(tempDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if m.Version != manifest.CurrentVersion {
		t.Errorf("Expected version %d, got %d", manifest.CurrentVersion, m.Version)
	}
	if m.Programs == nil {
		t.Errorf("Expected non-nil programs slice")
	}
}

func TestManifestSaveNilPrograms(t *testing.T) {
	tempDir := t.TempDir()
	m := &manifest.Manifest{
		Version:  1,
		Programs: nil,
	}
	if err := m.Save(tempDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := manifest.Load(tempDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Programs == nil || len(loaded.Programs) != 0 {
		t.Errorf("Expected empty programs slice, got %v", loaded.Programs)
	}
}
