package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPaths(t *testing.T) {
	dir, err := DefaultConfigDir()
	if err != nil || dir == "" {
		t.Fatalf("DefaultConfigDir returned error or empty: %v", err)
	}
	repo, err := DefaultRepoPath()
	if err != nil || repo == "" {
		t.Fatalf("DefaultRepoPath returned error or empty: %v", err)
	}
	filePath, err := DefaultConfigFilePath()
	if err != nil || filePath == "" {
		t.Fatalf("DefaultConfigFilePath returned error or empty: %v", err)
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	cfg := &Config{
		RemoteURL: "git@github.com:test/repo.git",
		RepoPath:  filepath.Join(tempDir, "repo"),
	}

	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("Failed to read saved config file: %v", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Errorf("Expected trailing newline in saved config file")
	}

	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loaded.RemoteURL != cfg.RemoteURL {
		t.Errorf("Expected RemoteURL %s, got %s", cfg.RemoteURL, loaded.RemoteURL)
	}
	if loaded.RepoPath != cfg.RepoPath {
		t.Errorf("Expected RepoPath %s, got %s", cfg.RepoPath, loaded.RepoPath)
	}
}

func TestLoadNonExistentUsesDefaults(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "does-not-exist.json")

	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Loading non-existent config should not error: %v", err)
	}
	if loaded.RepoPath == "" {
		t.Errorf("Expected default RepoPath, got empty")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "invalid.json")
	if err := os.WriteFile(cfgPath, []byte("invalid json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatalf("Expected error loading invalid JSON, got nil")
	}
}

func TestLoadEmptyRepoPathUsesDefault(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"remote_url":"git@github.com:test/repo.git","repo_path":""}`), 0644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}
	if loaded.RepoPath == "" {
		t.Errorf("Expected default RepoPath when empty in JSON, got empty string")
	}
	if loaded.RemoteURL != "git@github.com:test/repo.git" {
		t.Errorf("Expected RemoteURL to be preserved, got %s", loaded.RemoteURL)
	}
}

func TestSaveInvalidPath(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "not-a-directory")
	if err := os.WriteFile(filePath, []byte("file content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		RemoteURL: "git@github.com:test/repo.git",
	}
	// Attempting to create a directory inside a regular file will fail portably on both Windows and Linux.
	invalidPath := filepath.Join(filePath, "sub", "config.json")
	if err := cfg.Save(invalidPath); err == nil {
		t.Errorf("Expected Save to fail when parent path contains a regular file, got nil")
	}
}
