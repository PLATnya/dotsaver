package manifest

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ManifestFilename is the standard name of the repository manifest file.
const ManifestFilename = "dotsaver.json"

// CurrentVersion is the current schema version of the manifest.
const CurrentVersion = 1

// Target represents an OS-specific target path for a program's configuration.
type Target struct {
	OS    string `json:"os"`
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}

// Program represents an application and all its OS target configurations.
type Program struct {
	Name    string   `json:"name"`
	Targets []Target `json:"targets"`
}

// Manifest represents the root dotsaver.json configuration stored in the repository.
type Manifest struct {
	Version  int       `json:"version"`
	Programs []Program `json:"programs"`
}

// New creates and returns a new empty Manifest initialized with CurrentVersion.
func New() *Manifest {
	return &Manifest{
		Version:  CurrentVersion,
		Programs: []Program{},
	}
}

// Load reads and parses a Manifest from dotsaver.json within repoPath.
// If the file does not exist, a new empty Manifest is returned without error.
func Load(repoPath string) (*Manifest, error) {
	manifestPath := filepath.Join(repoPath, ManifestFilename)
	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
		return New(), nil
	} else if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	if m.Version == 0 {
		m.Version = CurrentVersion
	}
	if m.Programs == nil {
		m.Programs = []Program{}
	}
	for i := range m.Programs {
		if m.Programs[i].Targets == nil {
			m.Programs[i].Targets = []Target{}
		}
	}

	return &m, nil
}

// Save writes the Manifest to dotsaver.json within repoPath, creating directories if needed.
func (m *Manifest) Save(repoPath string) error {
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return err
	}
	if m.Programs == nil {
		m.Programs = []Program{}
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(filepath.Join(repoPath, ManifestFilename), data, 0644)
}

// FindProgram searches for a program by name (case-insensitive) and returns a pointer to it, or nil.
func (m *Manifest) FindProgram(name string) *Program {
	trimmed := strings.TrimSpace(name)
	for i := range m.Programs {
		if strings.EqualFold(m.Programs[i].Name, trimmed) {
			return &m.Programs[i]
		}
	}
	return nil
}

// AddProgram adds a new program to the manifest.
// Returns an error if the name is empty or if a program with the same name already exists.
func (m *Manifest) AddProgram(name string) (*Program, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("program name cannot be empty")
	}
	if m.FindProgram(name) != nil {
		return nil, errors.New("program already exists")
	}

	m.Programs = append(m.Programs, Program{
		Name:    name,
		Targets: []Target{},
	})
	return &m.Programs[len(m.Programs)-1], nil
}

// RemoveProgram removes a program by name (case-insensitive).
// Returns true if the program was found and removed, false otherwise.
func (m *Manifest) RemoveProgram(name string) bool {
	trimmed := strings.TrimSpace(name)
	for i, prog := range m.Programs {
		if strings.EqualFold(prog.Name, trimmed) {
			m.Programs = append(m.Programs[:i], m.Programs[i+1:]...)
			return true
		}
	}
	return false
}

// AddTarget adds an OS target configuration to an existing program.
// Returns an error if the program does not exist, OS is invalid, path is empty,
// or a target for the same OS and path already exists.
func (m *Manifest) AddTarget(progName string, target Target) error {
	prog := m.FindProgram(progName)
	if prog == nil {
		return errors.New("program not found")
	}

	target.OS = strings.ToLower(strings.TrimSpace(target.OS))
	target.Path = strings.TrimSpace(target.Path)

	if target.OS != "windows" && target.OS != "linux" {
		return errors.New("os must be 'windows' or 'linux'")
	}
	if target.Path == "" {
		return errors.New("path cannot be empty")
	}

	target.Alias = strings.TrimSpace(target.Alias)
	if target.Alias == "" {
		cleanPath := strings.ReplaceAll(target.Path, "\\", "/")
		cleanPath = path.Clean(cleanPath)
		target.Alias = path.Base(cleanPath)
		if target.Alias == "." || target.Alias == "/" || target.Alias == "" {
			target.Alias = prog.Name
		}
	}

	for _, existing := range prog.Targets {
		if existing.OS == target.OS && existing.Path == target.Path {
			return errors.New("target path already exists for this OS")
		}
	}

	prog.Targets = append(prog.Targets, target)
	return nil
}

// RemoveTarget removes a target configuration from a program matching OS and path.
// Returns true if found and removed, false otherwise.
func (m *Manifest) RemoveTarget(progName, osName, targetPath string) bool {
	prog := m.FindProgram(progName)
	if prog == nil {
		return false
	}

	osName = strings.ToLower(strings.TrimSpace(osName))
	targetPath = strings.TrimSpace(targetPath)

	for i, target := range prog.Targets {
		if target.OS == osName && target.Path == targetPath {
			prog.Targets = append(prog.Targets[:i], prog.Targets[i+1:]...)
			return true
		}
	}
	return false
}

// TargetsForOS returns a map of program name to slice of targets matching currentOS.
func (m *Manifest) TargetsForOS(currentOS string) map[string][]Target {
	currentOS = strings.ToLower(strings.TrimSpace(currentOS))
	res := make(map[string][]Target)

	for _, prog := range m.Programs {
		var matched []Target
		for _, target := range prog.Targets {
			if target.OS == currentOS {
				matched = append(matched, target)
			}
		}
		if len(matched) > 0 {
			res[prog.Name] = matched
		}
	}
	return res
}
