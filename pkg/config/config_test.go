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

	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("Config file does not exist on disk: %v", err)
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
	cfg := &Config{
		RemoteURL: "git@github.com:test/repo.git",
	}
	// Path with illegal filename characters or an invalid location
	invalidPath := filepath.Join("Z:", "nonexistent_drive_dotsaver_test", "sub", "config.json")
	if err := cfg.Save(invalidPath); err == nil {
		t.Errorf("Expected Save to fail on non-existent drive path, got nil")
	}
}
