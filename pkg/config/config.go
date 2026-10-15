package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// Config represents the local dotsaver configuration.
type Config struct {
	RemoteURL string `json:"remote_url"`
	RepoPath  string `json:"repo_path"`
}

// DefaultConfigDir returns the default directory for dotsaver configuration files.
func DefaultConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "dotsaver"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "dotsaver"), nil
}

// DefaultRepoPath returns the default directory where the dotfiles git repository is cached.
func DefaultRepoPath() (string, error) {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			return filepath.Join(localAppData, "dotsaver", "repo"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "dotsaver", "repo"), nil
}

// DefaultConfigFilePath returns the default path to the config.json file.
func DefaultConfigFilePath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads and parses a Config from the given file path.
// If the file does not exist, it returns a default Config with RepoPath populated.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		repoDefault, _ := DefaultRepoPath()
		return &Config{
			RepoPath: repoDefault,
		}, nil
	} else if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.RepoPath == "" {
		cfg.RepoPath, _ = DefaultRepoPath()
	}
	return &cfg, nil
}

// Save writes the Config as formatted JSON to the given path, creating parent directories if needed.
// It writes to a temporary file in the destination directory first and performs an atomic rename.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpFile, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}
