package syncer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var winEnvRegex = regexp.MustCompile(`%([a-zA-Z0-9_]+)%`)

// ExpandPath expands user home (~), Windows %VAR%, and Unix $VAR / ${VAR}
// environment variables in a file path, returning a clean platform-specific path.
// If an unset environment variable causes the path to evaluate to empty or ".", an error is returned.
func ExpandPath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", nil
	}

	// Expand Windows %VAR% syntax
	expanded := winEnvRegex.ReplaceAllStringFunc(rawPath, func(match string) string {
		varName := match[1 : len(match)-1]
		return os.Getenv(varName)
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

	cleaned := filepath.Clean(expanded)
	if (cleaned == "." || strings.TrimSpace(expanded) == "") && rawPath != "." && rawPath != "./" && rawPath != ".\\" {
		return "", fmt.Errorf("path %q evaluated to empty or working directory due to unset environment variable", rawPath)
	}

	return cleaned, nil
}
