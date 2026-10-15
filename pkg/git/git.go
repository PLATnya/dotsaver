package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitService struct{}

func New() *GitService {
	return &GitService{}
}

func (g *GitService) CheckGitInstalled() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return errors.New("git command not found in PATH. Please install Git")
	}
	return nil
}

func runCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		outTrimmed := strings.TrimSpace(output)
		if outTrimmed != "" {
			return output, fmt.Errorf("git %s failed: %w (output: %s)", strings.Join(args, " "), err, outTrimmed)
		}
		return output, fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(output), nil
}

func (g *GitService) EnsureRepo(remoteURL, repoPath string) error {
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			return err
		}

		// Try clone
		if remoteURL != "" {
			out, err := runCmd("", "clone", remoteURL, repoPath)
			if err == nil {
				return nil
			}
			// If clone failed because remote is empty or brand new, init locally
			_ = out
		}

		if _, err := runCmd(repoPath, "init", "-b", "main"); err != nil {
			// Older git may not support -b main
			if _, err := runCmd(repoPath, "init"); err != nil {
				return err
			}
			_, _ = runCmd(repoPath, "checkout", "-b", "main")
		}

		if remoteURL != "" {
			_, _ = runCmd(repoPath, "remote", "add", "origin", remoteURL)
		}
		return nil
	}

	// Repo exists, check remote
	if remoteURL != "" {
		remotes, _ := runCmd(repoPath, "remote", "get-url", "origin")
		if strings.TrimSpace(remotes) != remoteURL {
			_, err := runCmd(repoPath, "remote", "set-url", "origin", remoteURL)
			if err != nil {
				_, _ = runCmd(repoPath, "remote", "add", "origin", remoteURL)
			}
		}
	}
	return nil
}

func (g *GitService) Pull(repoPath string) (string, error) {
	// Detect default branch
	branch, err := runCmd(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "" || branch == "HEAD" {
		branch = "main"
	}
	return runCmd(repoPath, "pull", "--rebase", "origin", branch)
}

func (g *GitService) CommitAndPush(repoPath, message string) (string, error) {
	if _, err := runCmd(repoPath, "add", "-A"); err != nil {
		return "", err
	}

	// Check if there are changes to commit
	status, err := runCmd(repoPath, "status", "--porcelain")
	if err != nil {
		return "", err
	}

	var commitOut string
	if len(strings.TrimSpace(status)) > 0 {
		out, err := runCmd(repoPath, "commit", "-m", message)
		if err != nil {
			return out, err
		}
		commitOut = out
	}

	branch, err := runCmd(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "" || branch == "HEAD" {
		branch = "main"
	}

	pushOut, err := runCmd(repoPath, "push", "-u", "origin", branch)
	var combined string
	if commitOut != "" && pushOut != "" {
		combined = commitOut + "\n" + pushOut
	} else {
		combined = commitOut + pushOut
	}
	return strings.TrimSpace(combined), err
}

func (g *GitService) Status(repoPath string) (string, error) {
	return runCmd(repoPath, "status", "--short")
}

func (g *GitService) IsClean(repoPath string) (bool, error) {
	status, err := g.Status(repoPath)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(status) == "", nil
}
