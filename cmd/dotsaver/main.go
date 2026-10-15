package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/dotsaver/pkg/config"
	"github.com/user/dotsaver/pkg/tui"
)

// Version is the current version of dotsaver.
var Version = "0.1.0"

type appConfig struct {
	checkGit func() error
	runApp   func(model tea.Model) error
}

func defaultAppConfig() appConfig {
	return appConfig{
		checkGit: func() error {
			_, err := exec.LookPath("git")
			if err != nil {
				return errors.New("git executable not found in PATH")
			}
			return nil
		},
		runApp: func(m tea.Model) error {
			p := tea.NewProgram(m, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
}

func run(args []string, stdout, stderr io.Writer, appCfg appConfig) int {
	fs := flag.NewFlagSet("dotsaver", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var cfgPath string
	var showVersion bool
	var showHelp bool

	fs.StringVar(&cfgPath, "config", "", "Path to dotsaver configuration file")
	fs.StringVar(&cfgPath, "c", "", "Path to dotsaver configuration file (shorthand)")
	fs.BoolVar(&showVersion, "version", false, "Show dotsaver version")
	fs.BoolVar(&showVersion, "v", false, "Show dotsaver version (shorthand)")
	fs.BoolVar(&showHelp, "help", false, "Show help information")
	fs.BoolVar(&showHelp, "h", false, "Show help information (shorthand)")

	printHelp := func() {
		defaultCfg, _ := config.DefaultConfigFilePath()
		fmt.Fprintf(stdout, `dotsaver - Cross-platform TUI dotfile backup and synchronization tool

Usage:
  dotsaver [flags]

Flags:
  -c, --config <path>   Path to configuration file
                        (default: %s)
  -v, --version         Show dotsaver version
  -h, --help            Show this help message
`, defaultCfg)
	}

	fs.Usage = printHelp

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if showHelp {
		printHelp()
		return 0
	}

	if showVersion {
		fmt.Fprintf(stdout, "dotsaver v%s\n", Version)
		return 0
	}

	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: unexpected positional argument(s): %v\n", fs.Args())
		printHelp()
		return 2
	}

	// Pre-flight check: Verify git is installed and in PATH
	if err := appCfg.checkGit(); err != nil {
		fmt.Fprintln(stderr, "Error: 'git' executable not found in PATH.")
		fmt.Fprintln(stderr, "dotsaver requires Git to version and synchronize your dotfiles.")
		fmt.Fprintln(stderr, "Please install Git (https://git-scm.com/) and ensure it is available in your system PATH.")
		return 1
	}

	// Determine config file path if not provided
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultConfigFilePath()
		if err != nil {
			fmt.Fprintf(stderr, "Error: failed to determine default configuration path: %v\n", err)
			return 1
		}
	}

	// Initialize TUI application model
	model, err := tui.NewModel(cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: failed to initialize dotsaver: %v\n", err)
		return 1
	}

	// Launch Bubble Tea program with alternate screen buffer
	if err := appCfg.runApp(model); err != nil {
		fmt.Fprintf(stderr, "Error running dotsaver: %v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, defaultAppConfig()))
}
