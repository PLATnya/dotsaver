package syncer

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/dotsaver/pkg/git"
	"github.com/user/dotsaver/pkg/manifest"
)

// ConflictItem represents a mismatch between the repository cache and local configuration target.
type ConflictItem struct {
	Program    string
	TargetPath string
	RepoPath   string
}

// SyncReport contains results and diagnostics of a sync operation.
type SyncReport struct {
	Updated   []string
	BackedUp  []string
	Conflicts []ConflictItem
	Warnings  []string
}

// SaveReport contains results and diagnostics of a save operation.
type SaveReport struct {
	Saved    []string
	Warnings []string
}

// Syncer handles backing up, synchronizing, and conflict detection between local paths and repository cache.
type Syncer struct {
	Git *git.GitService
}

// NewSyncer creates a new Syncer instance with the provided GitService (or default git.New() if nil).
func NewSyncer(gitSvc *git.GitService) *Syncer {
	if gitSvc == nil {
		gitSvc = git.New()
	}
	return &Syncer{Git: gitSvc}
}

// RepoTargetDir returns the storage path in the repository cache for a specific target: data/<prog>/<os>/<alias>.
func (s *Syncer) RepoTargetDir(repoPath, programName, targetOS, alias string) string {
	if alias == "" {
		alias = programName
	}
	return filepath.Join(repoPath, "data", programName, targetOS, alias)
}

func resolveAlias(target manifest.Target, progName string) string {
	if target.Alias != "" {
		return target.Alias
	}
	clean := strings.ReplaceAll(target.Path, "\\", "/")
	clean = path.Clean(clean)
	alias := path.Base(clean)
	if alias == "." || alias == "/" || alias == "\\" || alias == "" {
		return progName
	}
	return alias
}

// Save copies local files/directories matching the current OS into the repository cache and updates dotsaver.json.
// Missing local source files record non-fatal warnings so the remaining targets can continue saving.
func (s *Syncer) Save(repoPath string, m *manifest.Manifest, currentOS string) (*SaveReport, error) {
	if m == nil {
		return nil, errors.New("manifest cannot be nil")
	}

	currentOS = strings.ToLower(strings.TrimSpace(currentOS))
	rep := &SaveReport{
		Saved:    []string{},
		Warnings: []string{},
	}

	targetsByProg := m.TargetsForOS(currentOS)
	for _, prog := range m.Programs {
		targets, ok := targetsByProg[prog.Name]
		if !ok {
			continue
		}

		for _, target := range targets {
			expanded, err := ExpandPath(target.Path)
			if err != nil {
				rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: cannot expand path %s: %v", prog.Name, target.Path, err))
				continue
			}

			if _, err := os.Stat(expanded); err != nil {
				if os.IsNotExist(err) {
					rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: path does not exist: %s", prog.Name, expanded))
				} else {
					rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: cannot access path %s: %v", prog.Name, expanded, err))
				}
				continue
			}

			alias := resolveAlias(target, prog.Name)
			dstRepoPath := s.RepoTargetDir(repoPath, prog.Name, currentOS, alias)
			if err := CopyEntry(expanded, dstRepoPath); err != nil {
				rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: failed to copy %s: %v", prog.Name, expanded, err))
				continue
			}
			rep.Saved = append(rep.Saved, fmt.Sprintf("%s -> %s", prog.Name, target.Path))
		}
	}

	if err := m.Save(repoPath); err != nil {
		return rep, fmt.Errorf("failed to save manifest: %w", err)
	}

	return rep, nil
}

// DetectConflicts checks targets for differences between repository cache and existing local files using SHA-256 comparison.
func (s *Syncer) DetectConflicts(repoPath string, m *manifest.Manifest, currentOS string) ([]ConflictItem, error) {
	conflicts := []ConflictItem{}
	if m == nil {
		return conflicts, nil
	}

	currentOS = strings.ToLower(strings.TrimSpace(currentOS))
	targetsByProg := m.TargetsForOS(currentOS)
	for _, prog := range m.Programs {
		targets, ok := targetsByProg[prog.Name]
		if !ok {
			continue
		}

		for _, target := range targets {
			alias := resolveAlias(target, prog.Name)
			repoEntry := s.RepoTargetDir(repoPath, prog.Name, currentOS, alias)
			if _, err := os.Stat(repoEntry); err != nil {
				continue
			}

			localPath, err := ExpandPath(target.Path)
			if err != nil {
				continue
			}

			if _, err := os.Stat(localPath); err != nil {
				// Local file doesn't exist, no conflict
				continue
			}

			identical, err := IsIdentical(repoEntry, localPath)
			if err != nil {
				continue
			}
			if !identical {
				conflicts = append(conflicts, ConflictItem{
					Program:    prog.Name,
					TargetPath: localPath,
					RepoPath:   repoEntry,
				})
			}
		}
	}
	return conflicts, nil
}

func createBackup(localPath string) (string, error) {
	bakPath := localPath + ".bak"
	if _, err := os.Stat(bakPath); err == nil {
		timestamp := time.Now().Format("20060102150405")
		bakPath = fmt.Sprintf("%s.bak.%s", localPath, timestamp)
		counter := 1
		for counter < 1000 {
			if _, err := os.Stat(bakPath); err != nil {
				break
			}
			bakPath = fmt.Sprintf("%s.bak.%s_%d", localPath, timestamp, counter)
			counter++
		}
	}
	err := os.Rename(localPath, bakPath)
	return bakPath, err
}

// Sync restores repository cache versions to local paths.
// If autoBackup is true, differing local targets are backed up to .bak (or .bak.YYYYMMDDHHMMSS if .bak exists) before being overwritten.
// If autoBackup is false, differing local targets are recorded as conflicts and skipped.
func (s *Syncer) Sync(repoPath string, m *manifest.Manifest, currentOS string, autoBackup bool) (*SyncReport, error) {
	if m == nil {
		return nil, errors.New("manifest cannot be nil")
	}

	currentOS = strings.ToLower(strings.TrimSpace(currentOS))
	rep := &SyncReport{
		Updated:   []string{},
		BackedUp:  []string{},
		Conflicts: []ConflictItem{},
		Warnings:  []string{},
	}

	targetsByProg := m.TargetsForOS(currentOS)
	for _, prog := range m.Programs {
		targets, ok := targetsByProg[prog.Name]
		if !ok {
			continue
		}

		for _, target := range targets {
			alias := resolveAlias(target, prog.Name)
			repoEntry := s.RepoTargetDir(repoPath, prog.Name, currentOS, alias)
			if _, err := os.Stat(repoEntry); err != nil {
				if os.IsNotExist(err) {
					rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: not found in repository cache (%s)", prog.Name, repoEntry))
				} else {
					rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: cannot access repository cache (%s): %v", prog.Name, repoEntry, err))
				}
				continue
			}

			localPath, err := ExpandPath(target.Path)
			if err != nil {
				rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: invalid path %s: %v", prog.Name, target.Path, err))
				continue
			}

			if _, err := os.Stat(localPath); err == nil {
				identical, err := IsIdentical(repoEntry, localPath)
				if err == nil && identical {
					continue // Already in sync
				}

				if !autoBackup {
					rep.Conflicts = append(rep.Conflicts, ConflictItem{
						Program:    prog.Name,
						TargetPath: localPath,
						RepoPath:   repoEntry,
					})
					continue
				}

				// Perform backup before overwriting
				bak, err := createBackup(localPath)
				if err != nil {
					rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: failed to create backup for %s: %v", prog.Name, localPath, err))
					continue
				}
				rep.BackedUp = append(rep.BackedUp, bak)
			}

			// Copy repo entry to local destination
			if err := CopyEntry(repoEntry, localPath); err != nil {
				rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: failed to restore to %s: %v", prog.Name, localPath, err))
				continue
			}
			rep.Updated = append(rep.Updated, localPath)
		}
	}

	return rep, nil
}
