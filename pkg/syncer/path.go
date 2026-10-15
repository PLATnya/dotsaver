package syncer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var winEnvRegex = regexp.MustCompile(`%([a-zA-Z0-9_]+)%`)

// ExpandPath expands user home (~), Windows %VAR%, and Unix $VAR / ${VAR}
// environment variables in a file path, returning a clean platform-specific path.
// Unset Windows variables are preserved safely, and unset Unix variables expand to empty.
func ExpandPath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", nil
	}

	// Expand Windows %VAR% syntax
	expanded := winEnvRegex.ReplaceAllStringFunc(rawPath, func(match string) string {
		varName := match[1 : len(match)-1]
		val := os.Getenv(varName)
		if val != "" {
			return val
		}
		return match
	})

	// Expand Unix $VAR and ${VAR}
	expanded = os.ExpandEnv(expanded)

	// Expand tilde ~
	if strings.HasPrefix(expanded, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if expanded == "~" {
			expanded = home
		} else if strings.HasPrefix(expanded, "~/") || strings.HasPrefix(expanded, `~\`) {
			expanded = filepath.Join(home, expanded[2:])
		}
	}

	return filepath.Clean(expanded), nil
}
