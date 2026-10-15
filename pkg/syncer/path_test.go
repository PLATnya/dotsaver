package syncer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPathTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home dir")
	}

	expanded, err := ExpandPath("~/testfile")
	if err != nil {
		t.Fatalf("ExpandPath returned error: %v", err)
	}

	expected := filepath.Join(home, "testfile")
	if expanded != expected {
		t.Errorf("Expected %s, got %s", expected, expanded)
	}
}

func TestExpandPathTildeOnly(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home dir")
	}

	expanded, err := ExpandPath("~")
	if err != nil {
		t.Fatalf("ExpandPath returned error: %v", err)
	}

	expected := filepath.Clean(home)
	if expanded != expected {
		t.Errorf("Expected %s, got %s", expected, expanded)
	}
}

func TestExpandPathEnvWindows(t *testing.T) {
	os.Setenv("DOTSAVER_TEST_VAR", "myfolder")
	defer os.Unsetenv("DOTSAVER_TEST_VAR")

	expanded, err := ExpandPath("C:/test/%DOTSAVER_TEST_VAR%/sub")
	if err != nil {
		t.Fatalf("ExpandPath failed: %v", err)
	}
	if !strings.Contains(expanded, "myfolder") {
		t.Errorf("Expected 'myfolder' in expanded path, got: %s", expanded)
	}
}

func TestExpandPathEnvUnix(t *testing.T) {
	os.Setenv("DOTSAVER_UNIX_VAR", "configs")
	defer os.Unsetenv("DOTSAVER_UNIX_VAR")

	expanded, err := ExpandPath("/etc/$DOTSAVER_UNIX_VAR/app")
	if err != nil {
		t.Fatalf("ExpandPath failed: %v", err)
	}
	if !strings.Contains(expanded, "configs") {
		t.Errorf("Expected 'configs' in expanded path, got: %s", expanded)
	}

	// Also test ${VAR} curly braces syntax
	expandedCurly, err := ExpandPath("/etc/${DOTSAVER_UNIX_VAR}/app")
	if err != nil {
		t.Fatalf("ExpandPath with curly syntax failed: %v", err)
	}
	if !strings.Contains(expandedCurly, "configs") {
		t.Errorf("Expected 'configs' in expandedCurly path, got: %s", expandedCurly)
	}
}

func TestExpandPathUnsetEnvSafe(t *testing.T) {
	expanded, err := ExpandPath("/test/%UNSET_VAR_XYZ%/app")
	if err != nil {
		t.Fatalf("ExpandPath on unset env failed: %v", err)
	}
	if expanded == "" {
		t.Errorf("Expected non-empty result on unset env, got empty")
	}
}

func TestExpandPathEmpty(t *testing.T) {
	expanded, err := ExpandPath("")
	if err != nil {
		t.Fatalf("ExpandPath on empty string returned error: %v", err)
	}
	if expanded != "" {
		t.Errorf("Expected empty string, got: %q", expanded)
	}

	expandedSpace, err := ExpandPath("   ")
	if err != nil {
		t.Fatalf("ExpandPath on whitespace string returned error: %v", err)
	}
	if expandedSpace != "" {
		t.Errorf("Expected empty string, got: %q", expandedSpace)
	}
}
