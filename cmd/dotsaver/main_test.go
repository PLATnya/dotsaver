package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunVersion(t *testing.T) {
	for _, flag := range []string{"-version", "--version", "-v"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{flag}, &stdout, &stderr, defaultAppConfig())
			if code != 0 {
				t.Fatalf("Expected exit code 0 for flag %s, got %d", flag, code)
			}
			if !strings.Contains(stdout.String(), "dotsaver v"+Version) {
				t.Errorf("Expected output to contain version, got: %s", stdout.String())
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	for _, flag := range []string{"-help", "--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{flag}, &stdout, &stderr, defaultAppConfig())
			if code != 0 {
				t.Fatalf("Expected exit code 0 for flag %s, got %d", flag, code)
			}
			if !strings.Contains(stdout.String(), "dotsaver - Cross-platform TUI dotfile backup") {
				t.Errorf("Expected output to contain usage info, got: %s", stdout.String())
			}
		})
	}
}

func TestRunInvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--invalid-flag"}, &stdout, &stderr, defaultAppConfig())
	if code != 2 {
		t.Fatalf("Expected exit code 2 for invalid flag, got %d", code)
	}
}

func TestRunPositionalArgsRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"extra-arg"}, &stdout, &stderr, defaultAppConfig())
	if code != 2 {
		t.Fatalf("Expected exit code 2 for positional arguments, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unexpected positional argument") {
		t.Errorf("Expected stderr to mention unexpected positional argument, got: %s", stderr.String())
	}
}

func TestRunGitMissing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	mockCfg := appConfig{
		checkGit: func() error {
			return errors.New("git not found")
		},
		runApp: func(m tea.Model) error {
			return nil
		},
	}

	code := run([]string{}, &stdout, &stderr, mockCfg)
	if code != 1 {
		t.Fatalf("Expected exit code 1 when git is missing, got %d", code)
	}
	if !strings.Contains(stderr.String(), "'git' executable not found in PATH") {
		t.Errorf("Expected user-friendly git missing error message, got: %s", stderr.String())
	}
}

func TestRunWithConfigFlag(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "custom_config.json")

	var launched bool
	mockCfg := appConfig{
		checkGit: func() error {
			return nil
		},
		runApp: func(m tea.Model) error {
			launched = true
			return nil
		},
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", cfgPath}, &stdout, &stderr, mockCfg)
	if code != 0 {
		t.Fatalf("Expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !launched {
		t.Errorf("Expected runApp to have been called")
	}
}

func TestRunAppError(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "custom_config.json")

	mockCfg := appConfig{
		checkGit: func() error {
			return nil
		},
		runApp: func(m tea.Model) error {
			return errors.New("terminal failure")
		},
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-c", cfgPath}, &stdout, &stderr, mockCfg)
	if code != 1 {
		t.Fatalf("Expected exit code 1 on app run failure, got %d", code)
	}
	if !strings.Contains(stderr.String(), "terminal failure") {
		t.Errorf("Expected stderr to contain run error, got: %s", stderr.String())
	}
}
