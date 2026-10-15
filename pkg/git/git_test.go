package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckGitInstalled(t *testing.T) {
	svc := New()
	if err := svc.CheckGitInstalled(); err != nil {
		t.Fatalf("Expected git to be installed, got: %v", err)
	}
}

func TestGitServiceWithLocalBareRepo(t *testing.T) {
	svc := New()
	if err := svc.CheckGitInstalled(); err != nil {
		t.Skip("Git not installed on system, skipping")
	}

	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "remote.git")
	clientRepo := filepath.Join(tempDir, "client")

	// Init bare repo
	cmd := exec.Command("git", "init", "--bare", bareRepo)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	// EnsureRepo (clones or initializes)
	err := svc.EnsureRepo(bareRepo, clientRepo)
	if err != nil {
		t.Fatalf("EnsureRepo failed: %v", err)
	}

	// Initial repo should be clean
	clean, err := svc.IsClean(clientRepo)
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}
	if !clean {
		t.Errorf("Expected initial repo to be clean, got clean=%v", clean)
	}

	// Create a dummy file in clientRepo
	testFile := filepath.Join(clientRepo, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Check status when dirty
	clean, err = svc.IsClean(clientRepo)
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}
	if clean {
		t.Errorf("Expected repo to not be clean after writing file")
	}

	status, err := svc.Status(clientRepo)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if !strings.Contains(status, "test.txt") {
		t.Errorf("Expected status to mention test.txt, got: %s", status)
	}

	// Commit and push
	out, err := svc.CommitAndPush(clientRepo, "initial commit")
	if err != nil {
		t.Fatalf("CommitAndPush failed: %v\nOutput: %s", err, out)
	}

	// Check status clean after commit and push
	clean, err = svc.IsClean(clientRepo)
	if err != nil || !clean {
		t.Errorf("Expected repo clean after commit and push, clean=%v, err=%v", clean, err)
	}

	// Clone in second client and pull
	clientRepo2 := filepath.Join(tempDir, "client2")
	err = svc.EnsureRepo(bareRepo, clientRepo2)
	if err != nil {
		t.Fatalf("EnsureRepo client2 failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(clientRepo2, "test.txt")); err != nil {
		t.Errorf("test.txt was not pulled to client2: %v", err)
	}

	// Commit from client2 and pull into client1
	testFile2 := filepath.Join(clientRepo2, "test2.txt")
	if err := os.WriteFile(testFile2, []byte("test2 content"), 0644); err != nil {
		t.Fatalf("Failed to write test2 file: %v", err)
	}
	out, err = svc.CommitAndPush(clientRepo2, "second commit")
	if err != nil {
		t.Fatalf("CommitAndPush client2 failed: %v\nOutput: %s", err, out)
	}

	pullOut, err := svc.Pull(clientRepo)
	if err != nil {
		t.Fatalf("Pull client1 failed: %v\nOutput: %s", err, pullOut)
	}

	if _, err := os.Stat(filepath.Join(clientRepo, "test2.txt")); err != nil {
		t.Errorf("test2.txt was not pulled to client1: %v", err)
	}
}

func TestEnsureRepoLocalOnly(t *testing.T) {
	svc := New()
	if err := svc.CheckGitInstalled(); err != nil {
		t.Skip("Git not installed on system, skipping")
	}

	tempDir := t.TempDir()
	localRepo := filepath.Join(tempDir, "local_only")

	err := svc.EnsureRepo("", localRepo)
	if err != nil {
		t.Fatalf("EnsureRepo local only failed: %v", err)
	}

	gitDir := filepath.Join(localRepo, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Fatalf("Expected .git directory at %s", gitDir)
	}

	// EnsureRepo again on existing repo should succeed
	err = svc.EnsureRepo("", localRepo)
	if err != nil {
		t.Fatalf("EnsureRepo on existing local repo failed: %v", err)
	}
}

func TestEnsureRepoUpdateRemoteURL(t *testing.T) {
	svc := New()
	if err := svc.CheckGitInstalled(); err != nil {
		t.Skip("Git not installed on system, skipping")
	}

	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	remote1 := "https://example.com/repo1.git"
	remote2 := "https://example.com/repo2.git"

	err := svc.EnsureRepo(remote1, repoPath)
	if err != nil {
		t.Fatalf("EnsureRepo failed: %v", err)
	}

	// Change remote
	err = svc.EnsureRepo(remote2, repoPath)
	if err != nil {
		t.Fatalf("EnsureRepo with new remote failed: %v", err)
	}

	out, err := runCmd(repoPath, "remote", "get-url", "origin")
	if err != nil {
		t.Fatalf("Failed to get remote origin: %v", err)
	}
	if strings.TrimSpace(out) != remote2 {
		t.Errorf("Expected remote origin %q, got %q", remote2, out)
	}
}

func TestCommitAndPushNoChanges(t *testing.T) {
	svc := New()
	if err := svc.CheckGitInstalled(); err != nil {
		t.Skip("Git not installed on system, skipping")
	}

	tempDir := t.TempDir()
	bareRepo := filepath.Join(tempDir, "remote.git")
	clientRepo := filepath.Join(tempDir, "client")

	cmd := exec.Command("git", "init", "--bare", bareRepo)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	err := svc.EnsureRepo(bareRepo, clientRepo)
	if err != nil {
		t.Fatalf("EnsureRepo failed: %v", err)
	}

	// Create and commit first file so branch exists
	testFile := filepath.Join(clientRepo, "init.txt")
	if err := os.WriteFile(testFile, []byte("init"), 0644); err != nil {
		t.Fatalf("Failed to write init file: %v", err)
	}
	if _, err := svc.CommitAndPush(clientRepo, "init"); err != nil {
		t.Fatalf("CommitAndPush init failed: %v", err)
	}

	// Now call CommitAndPush with no changes
	out, err := svc.CommitAndPush(clientRepo, "nothing to commit")
	if err != nil {
		t.Fatalf("CommitAndPush without changes should succeed, got err: %v, out: %s", err, out)
	}
}

func TestErrorFormattingCapturesStderr(t *testing.T) {
	svc := New()
	if err := svc.CheckGitInstalled(); err != nil {
		t.Skip("Git not installed on system, skipping")
	}

	tempDir := t.TempDir()
	nonExistentDir := filepath.Join(tempDir, "does-not-exist")

	// Calling status on a non-existent directory / non-repo should fail with actionable message
	_, err := svc.Status(nonExistentDir)
	if err == nil {
		t.Fatal("Expected error when running status on non-existent directory, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "git status") && !strings.Contains(errMsg, "failed") {
		t.Errorf("Expected error to contain command failure info, got: %s", errMsg)
	}
}
