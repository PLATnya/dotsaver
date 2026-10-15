package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/user/dotsaver/pkg/config"
	"github.com/user/dotsaver/pkg/git"
	"github.com/user/dotsaver/pkg/manifest"
	"github.com/user/dotsaver/pkg/syncer"
	"github.com/user/dotsaver/pkg/tui"
)

func setupTestGitIdentity(t *testing.T, repoDirs ...string) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "dotsaver-e2e")
	t.Setenv("GIT_AUTHOR_EMAIL", "e2e@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "dotsaver-e2e")
	t.Setenv("GIT_COMMITTER_EMAIL", "e2e@example.com")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.autocrlf")
	t.Setenv("GIT_CONFIG_VALUE_0", "false")

	for _, dir := range repoDirs {
		cmd1 := exec.Command("git", "config", "user.name", "dotsaver-e2e")
		cmd1.Dir = dir
		_ = cmd1.Run()
		cmd2 := exec.Command("git", "config", "user.email", "e2e@example.com")
		cmd2.Dir = dir
		_ = cmd2.Run()
		cmd3 := exec.Command("git", "config", "core.autocrlf", "false")
		cmd3.Dir = dir
		_ = cmd3.Run()
	}
}

func normalizeEOL(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func TestEndToEndLifecycle(t *testing.T) {
	gitSvc := git.New()
	if err := gitSvc.CheckGitInstalled(); err != nil {
		t.Skip("Git is not installed on system, skipping e2e test")
	}

	tempDir := t.TempDir()
	setupTestGitIdentity(t)

	// Step 1: Initialize local bare git repository as mock remote origin
	bareRemote := filepath.Join(tempDir, "remote.git")
	cmd := exec.Command("git", "init", "--bare", bareRemote)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to initialize bare remote repo: %v (output: %s)", err, string(out))
	}

	// Step 2: Initialize dotsaver config for Client 1 with this remote
	client1Dir := filepath.Join(tempDir, "client1")
	client1Repo := filepath.Join(client1Dir, "repo")
	client1CfgPath := filepath.Join(client1Dir, "config.json")

	cfg1 := &config.Config{
		RemoteURL: bareRemote,
		RepoPath:  client1Repo,
	}
	if err := cfg1.Save(client1CfgPath); err != nil {
		t.Fatalf("Client 1 failed to save config: %v", err)
	}

	loadedCfg1, err := config.Load(client1CfgPath)
	if err != nil || loadedCfg1.RemoteURL != bareRemote {
		t.Fatalf("Failed to reload client 1 config: %v", err)
	}

	if err := gitSvc.EnsureRepo(bareRemote, client1Repo); err != nil {
		t.Fatalf("Client 1 EnsureRepo failed: %v", err)
	}
	setupTestGitIdentity(t, client1Repo)

	// Step 3: Add program "demoapp" with targets (single file & directory)
	client1Targets := filepath.Join(client1Dir, "dotfiles")
	if err := os.MkdirAll(client1Targets, 0755); err != nil {
		t.Fatalf("Failed to create client 1 targets directory: %v", err)
	}

	// 3a: Single file target
	fileTarget := filepath.Join(client1Targets, "demoapp.conf")
	confInitial := []byte("theme = dark\nversion = 1.0.0\n")
	if err := os.WriteFile(fileTarget, confInitial, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// 3b: Directory target with nested subfiles
	dirTarget := filepath.Join(client1Targets, "plugins")
	nestedDir := filepath.Join(dirTarget, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create plugins nested dir: %v", err)
	}
	pluginAMain := filepath.Join(dirTarget, "main.lua")
	pluginBNested := filepath.Join(nestedDir, "helper.lua")
	if err := os.WriteFile(pluginAMain, []byte("return { enabled = true }"), 0644); err != nil {
		t.Fatalf("Failed to write plugin main.lua: %v", err)
	}
	if err := os.WriteFile(pluginBNested, []byte("function helper() end"), 0644); err != nil {
		t.Fatalf("Failed to write plugin helper.lua: %v", err)
	}

	// Add program and targets to manifest
	m1, err := manifest.Load(client1Repo)
	if err != nil {
		t.Fatalf("Failed to load client 1 manifest: %v", err)
	}
	if _, err := m1.AddProgram("demoapp"); err != nil {
		t.Fatalf("Failed to add program demoapp: %v", err)
	}
	if err := m1.AddTarget("demoapp", manifest.Target{
		OS:    runtime.GOOS,
		Path:  fileTarget,
		Alias: "demoapp.conf",
	}); err != nil {
		t.Fatalf("Failed to add file target: %v", err)
	}
	if err := m1.AddTarget("demoapp", manifest.Target{
		OS:    runtime.GOOS,
		Path:  dirTarget,
		Alias: "plugins",
	}); err != nil {
		t.Fatalf("Failed to add directory target: %v", err)
	}

	// Step 4: Call Save -> copies files to repo cache, commits, and pushes to remote
	syncEngine := syncer.NewSyncer(gitSvc)
	saveRep, err := syncEngine.Save(client1Repo, m1, runtime.GOOS)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if len(saveRep.Saved) != 2 {
		t.Fatalf("Expected 2 saved targets, got %d (%v)", len(saveRep.Saved), saveRep.Saved)
	}
	if len(saveRep.Warnings) != 0 {
		t.Fatalf("Expected 0 warnings on Save, got %d (%v)", len(saveRep.Warnings), saveRep.Warnings)
	}

	// Verify cached files exist in repository cache directory
	cachedFile := filepath.Join(client1Repo, "data", "demoapp", runtime.GOOS, "demoapp.conf")
	if data, err := os.ReadFile(cachedFile); err != nil || string(data) != string(confInitial) {
		t.Fatalf("Cached file not found or corrupted: %v", err)
	}
	cachedPluginA := filepath.Join(client1Repo, "data", "demoapp", runtime.GOOS, "plugins", "main.lua")
	if _, err := os.Stat(cachedPluginA); err != nil {
		t.Fatalf("Cached directory item main.lua not found: %v", err)
	}
	cachedPluginB := filepath.Join(client1Repo, "data", "demoapp", runtime.GOOS, "plugins", "nested", "helper.lua")
	if _, err := os.Stat(cachedPluginB); err != nil {
		t.Fatalf("Cached directory item helper.lua not found: %v", err)
	}

	// Commit and push to remote bare repo
	commitOut, err := gitSvc.CommitAndPush(client1Repo, "dotsaver: initial backup")
	if err != nil {
		t.Fatalf("CommitAndPush failed: %v\nOutput: %s", err, commitOut)
	}

	clean, err := gitSvc.IsClean(client1Repo)
	if err != nil || !clean {
		t.Fatalf("Expected client 1 repo to be clean after push, clean=%v, err=%v", clean, err)
	}

	// Step 5: Second client Sync: verifies files pulled from remote and restored to new target location
	client2Dir := filepath.Join(tempDir, "client2")
	client2Repo := filepath.Join(client2Dir, "repo")
	client2CfgPath := filepath.Join(client2Dir, "config.json")

	cfg2 := &config.Config{
		RemoteURL: bareRemote,
		RepoPath:  client2Repo,
	}
	if err := cfg2.Save(client2CfgPath); err != nil {
		t.Fatalf("Client 2 failed to save config: %v", err)
	}

	// Clone remote repo to client 2
	if err := gitSvc.EnsureRepo(bareRemote, client2Repo); err != nil {
		t.Fatalf("Client 2 EnsureRepo (clone) failed: %v", err)
	}
	setupTestGitIdentity(t, client2Repo)

	m2, err := manifest.Load(client2Repo)
	if err != nil {
		t.Fatalf("Client 2 failed to load manifest: %v", err)
	}
	if m2.FindProgram("demoapp") == nil {
		t.Fatalf("Client 2 manifest does not contain program demoapp")
	}

	// Map client 2 targets to client 2's local machine filesystem
	client2Targets := filepath.Join(client2Dir, "restored_dotfiles")
	client2File := filepath.Join(client2Targets, "demoapp.conf")
	client2DirTarget := filepath.Join(client2Targets, "plugins")

	prog2 := m2.FindProgram("demoapp")
	for i := range prog2.Targets {
		if prog2.Targets[i].Alias == "demoapp.conf" {
			prog2.Targets[i].Path = client2File
		} else if prog2.Targets[i].Alias == "plugins" {
			prog2.Targets[i].Path = client2DirTarget
		}
	}

	// Ensure destination does not exist prior to sync
	if _, err := os.Stat(client2File); !os.IsNotExist(err) {
		t.Fatalf("Expected client2File to not exist prior to Sync")
	}

	// Run Sync on Client 2
	syncRep2, err := syncEngine.Sync(client2Repo, m2, runtime.GOOS, false)
	if err != nil {
		t.Fatalf("Client 2 Sync failed: %v", err)
	}
	if len(syncRep2.Updated) != 2 {
		t.Fatalf("Expected 2 restored targets on Client 2, got %d (%v)", len(syncRep2.Updated), syncRep2.Updated)
	}
	if len(syncRep2.Conflicts) != 0 {
		t.Fatalf("Expected 0 conflicts on fresh sync, got %d", len(syncRep2.Conflicts))
	}

	// Verify restored files match original content
	restoredData, err := os.ReadFile(client2File)
	if err != nil || normalizeEOL(string(restoredData)) != normalizeEOL(string(confInitial)) {
		t.Fatalf("Restored client 2 config content mismatch: %v, content: %s", err, string(restoredData))
	}
	restoredPluginA, err := os.ReadFile(filepath.Join(client2DirTarget, "main.lua"))
	if err != nil || normalizeEOL(string(restoredPluginA)) != "return { enabled = true }" {
		t.Fatalf("Restored client 2 plugin main.lua mismatch: %v", err)
	}
	restoredPluginB, err := os.ReadFile(filepath.Join(client2DirTarget, "nested", "helper.lua"))
	if err != nil || normalizeEOL(string(restoredPluginB)) != "function helper() end" {
		t.Fatalf("Restored client 2 plugin helper.lua mismatch: %v", err)
	}

	// Running sync again should be a no-op (already identical)
	syncRepNoOp, err := syncEngine.Sync(client2Repo, m2, runtime.GOOS, false)
	if err != nil || len(syncRepNoOp.Updated) != 0 {
		t.Fatalf("Expected 0 updates on repeated Sync, got %d", len(syncRepNoOp.Updated))
	}

	// Step 6: Modify local target file to simulate conflict & verify conflict detection
	confModified := []byte("theme = custom_light\nversion = 2.0.0\n")
	if err := os.WriteFile(client2File, confModified, 0644); err != nil {
		t.Fatalf("Failed to overwrite client 2 config for conflict test: %v", err)
	}
	pluginAModified := []byte("return { enabled = false, user_override = true }")
	if err := os.WriteFile(filepath.Join(client2DirTarget, "main.lua"), pluginAModified, 0644); err != nil {
		t.Fatalf("Failed to overwrite client 2 plugin for conflict test: %v", err)
	}

	// DetectConflicts should report both targets as conflicts
	conflicts, err := syncEngine.DetectConflicts(client2Repo, m2, runtime.GOOS)
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}
	if len(conflicts) != 2 {
		t.Fatalf("Expected 2 conflicts, got %d (%v)", len(conflicts), conflicts)
	}

	// Sync with autoBackup = false should list conflicts and NOT overwrite files
	syncRepNoBak, err := syncEngine.Sync(client2Repo, m2, runtime.GOOS, false)
	if err != nil {
		t.Fatalf("Sync without autoBackup failed: %v", err)
	}
	if len(syncRepNoBak.Conflicts) != 2 {
		t.Fatalf("Expected 2 conflicts reported, got %d", len(syncRepNoBak.Conflicts))
	}
	if len(syncRepNoBak.BackedUp) != 0 {
		t.Fatalf("Expected 0 backed up files when autoBackup is false, got %d", len(syncRepNoBak.BackedUp))
	}
	// Verify local files still retain user changes
	if data, _ := os.ReadFile(client2File); normalizeEOL(string(data)) != normalizeEOL(string(confModified)) {
		t.Fatalf("Local file was prematurely overwritten without autoBackup")
	}

	// Step 7: Run Sync with autoBackup -> verifies .bak created and repo version cleanly restored
	syncRepBak, err := syncEngine.Sync(client2Repo, m2, runtime.GOOS, true)
	if err != nil {
		t.Fatalf("Sync with autoBackup failed: %v", err)
	}
	if len(syncRepBak.BackedUp) != 2 {
		t.Fatalf("Expected 2 backed up targets, got %d (%v)", len(syncRepBak.BackedUp), syncRepBak.BackedUp)
	}
	if len(syncRepBak.Updated) != 2 {
		t.Fatalf("Expected 2 restored targets, got %d", len(syncRepBak.Updated))
	}

	// Verify .bak file for single file exists and contains user's modified content
	bakFile := client2File + ".bak"
	bakData, err := os.ReadFile(bakFile)
	if err != nil {
		t.Fatalf("Expected backup file %s not found: %v", bakFile, err)
	}
	if normalizeEOL(string(bakData)) != normalizeEOL(string(confModified)) {
		t.Fatalf("Backup file content mismatch: expected '%s', got '%s'", string(confModified), string(bakData))
	}

	// Verify .bak directory exists and contains user's modified plugin content
	bakDir := client2DirTarget + ".bak"
	bakPluginA, err := os.ReadFile(filepath.Join(bakDir, "main.lua"))
	if err != nil {
		t.Fatalf("Expected backup directory plugin file not found: %v", err)
	}
	if normalizeEOL(string(bakPluginA)) != normalizeEOL(string(pluginAModified)) {
		t.Fatalf("Backup directory content mismatch: expected '%s', got '%s'", string(pluginAModified), string(bakPluginA))
	}

	// Verify restored target file was overwritten cleanly back to repo version
	restoredAfterBak, err := os.ReadFile(client2File)
	if err != nil || normalizeEOL(string(restoredAfterBak)) != normalizeEOL(string(confInitial)) {
		t.Fatalf("Restored file does not match repository content: expected '%s', got '%s'", string(confInitial), string(restoredAfterBak))
	}
	restoredPluginAfterBak, err := os.ReadFile(filepath.Join(client2DirTarget, "main.lua"))
	if err != nil || normalizeEOL(string(restoredPluginAfterBak)) != "return { enabled = true }" {
		t.Fatalf("Restored plugin main.lua does not match repo version: got '%s'", string(restoredPluginAfterBak))
	}

	// Conflicts after sync should now be 0
	conflictsAfter, err := syncEngine.DetectConflicts(client2Repo, m2, runtime.GOOS)
	if err != nil || len(conflictsAfter) != 0 {
		t.Fatalf("Expected 0 conflicts after clean sync restore, got %d (err: %v)", len(conflictsAfter), err)
	}

	// Step 8: Upstream roundtrip verification (Client 2 updates & pushes, Client 1 pulls & syncs)
	confSolarized := []byte("theme = solarized\nversion = 1.1.0\n")
	if err := os.WriteFile(client2File, confSolarized, 0644); err != nil {
		t.Fatalf("Client 2 failed to update file: %v", err)
	}

	if _, err := syncEngine.Save(client2Repo, m2, runtime.GOOS); err != nil {
		t.Fatalf("Client 2 Save failed: %v", err)
	}
	if out, err := gitSvc.CommitAndPush(client2Repo, "dotsaver: update theme to solarized"); err != nil {
		t.Fatalf("Client 2 CommitAndPush failed: %v\nOutput: %s", err, out)
	}

	// Client 1 pulls from remote
	if pullOut, err := gitSvc.Pull(client1Repo); err != nil {
		t.Fatalf("Client 1 Pull failed: %v\nOutput: %s", err, pullOut)
	}

	// Client 1 syncs
	if _, err := syncEngine.Sync(client1Repo, m1, runtime.GOOS, true); err != nil {
		t.Fatalf("Client 1 Sync failed: %v", err)
	}

	// Client 1's local file should now match solarized theme
	client1Restored, err := os.ReadFile(fileTarget)
	if err != nil || normalizeEOL(string(client1Restored)) != normalizeEOL(string(confSolarized)) {
		t.Fatalf("Client 1 did not receive upstream update: expected '%s', got '%s'", string(confSolarized), string(client1Restored))
	}

	// Step 9: TUI Model Integration Smoke Test
	model, err := tui.NewModel(client1CfgPath)
	if err != nil {
		t.Fatalf("Failed to initialize TUI Model with client 1 config: %v", err)
	}
	view := model.View()
	if !strings.Contains(view, "dotsaver") {
		t.Errorf("Expected 'dotsaver' title in view output, got: %s", view)
	}
	if !strings.Contains(view, "demoapp") {
		t.Errorf("Expected 'demoapp' in view output, got: %s", view)
	}
}

func TestCLIBinaryBuildAndRun(t *testing.T) {
	tempDir := t.TempDir()
	binName := "dotsaver"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(tempDir, binName)

	// Build binary
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/dotsaver")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\nOutput: %s", err, string(out))
	}

	// Test --version
	verCmd := exec.Command(binPath, "--version")
	out, err := verCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v\nOutput: %s", err, string(out))
	}
	if !strings.Contains(string(out), "dotsaver v") {
		t.Errorf("Expected version in output, got: %s", string(out))
	}

	// Test -v shorthand
	verCmdShort := exec.Command(binPath, "-v")
	out, err = verCmdShort.CombinedOutput()
	if err != nil {
		t.Fatalf("-v failed: %v\nOutput: %s", err, string(out))
	}
	if !strings.Contains(string(out), "dotsaver v") {
		t.Errorf("Expected version in output, got: %s", string(out))
	}

	// Test --help
	helpCmd := exec.Command(binPath, "--help")
	out, err = helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\nOutput: %s", err, string(out))
	}
	if !strings.Contains(string(out), "dotsaver - Cross-platform TUI dotfile backup") {
		t.Errorf("Expected help in output, got: %s", string(out))
	}

	// Test -h shorthand
	helpCmdShort := exec.Command(binPath, "-h")
	out, err = helpCmdShort.CombinedOutput()
	if err != nil {
		t.Fatalf("-h failed: %v\nOutput: %s", err, string(out))
	}
	if !strings.Contains(string(out), "dotsaver - Cross-platform TUI dotfile backup") {
		t.Errorf("Expected help in output, got: %s", string(out))
	}

	// Test invalid flag exits non-zero
	invalidCmd := exec.Command(binPath, "--nonexistent-flag")
	if err := invalidCmd.Run(); err == nil {
		t.Errorf("Expected error exit for invalid flag, got success")
	}
}
