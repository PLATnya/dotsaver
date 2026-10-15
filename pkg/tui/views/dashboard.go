package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgramSummary provides high-level metrics for a program in the dashboard.
type ProgramSummary struct {
	Name        string
	TargetCount int
	Paths       []string
}

// DashboardData contains all data needed to render the Dashboard view.
type DashboardData struct {
	RemoteURL     string
	RepoPath      string
	GitBranch     string
	GitStatus     string
	IsGitClean    bool
	CurrentOS     string
	Programs      []ProgramSummary
	TotalPrograms int
	TotalTargets  int
}

var (
	dashCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444455")).
			Padding(1, 2)

	dashHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D2FF")).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#AAAAAA")).
			Width(16)

	valStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	badgeClean = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#04B575")).
			Padding(0, 1)

	badgeDirty = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFB86C")).
			Padding(0, 1)

	badgeUninit = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#888888")).
			Padding(0, 1)

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777"))
)

// RenderDashboard generates the dashboard view string based on the provided DashboardData and terminal width.
func RenderDashboard(data DashboardData, width int) string {
	contentWidth := width - 4
	if contentWidth < 50 {
		contentWidth = 76
	}

	// 1. Repository Status Section
	var repoStatusBadge string
	if data.GitStatus == "Clean" {
		repoStatusBadge = badgeClean.Render("CLEAN")
	} else if data.GitStatus == "Uninitialized" || data.GitStatus == "Not a git repository" {
		repoStatusBadge = badgeUninit.Render("NO GIT")
	} else {
		repoStatusBadge = badgeDirty.Render("CHANGES PENDING")
	}

	remoteDisplay := data.RemoteURL
	if remoteDisplay == "" {
		remoteDisplay = mutedStyle.Render("(none configured - local cache only)")
	}

	branchDisplay := data.GitBranch
	if branchDisplay == "" {
		branchDisplay = "main"
	}

	repoInfo := strings.Join([]string{
		dashHeaderStyle.Render("Repository & Status"),
		fmt.Sprintf("%s %s", labelStyle.Render("Remote URL:"), valStyle.Render(remoteDisplay)),
		fmt.Sprintf("%s %s", labelStyle.Render("Cache Path:"), valStyle.Render(data.RepoPath)),
		fmt.Sprintf("%s %s", labelStyle.Render("Current OS:"), valStyle.Render(strings.ToUpper(data.CurrentOS))),
		fmt.Sprintf("%s %s (branch: %s)", labelStyle.Render("Git Status:"), repoStatusBadge, valStyle.Render(branchDisplay)),
	}, "\n")

	repoCard := dashCardStyle.Width(contentWidth).Render(repoInfo)

	// 2. Dotfiles Summary Section
	var dotfilesContent string
	if len(data.Programs) == 0 {
		dotfilesContent = strings.Join([]string{
			dashHeaderStyle.Render("Configured Dotfiles"),
			mutedStyle.Render("No programs registered yet."),
			mutedStyle.Render("Press [2] or 'a' to navigate to Programs and register your first dotfile!"),
		}, "\n")
	} else {
		var lines []string
		lines = append(lines, dashHeaderStyle.Render(fmt.Sprintf("Configured Dotfiles (%d programs, %d targets for %s)",
			data.TotalPrograms, data.TotalTargets, data.CurrentOS)))

		progHeader := fmt.Sprintf("  %-16s %-14s %s", "PROGRAM", "TARGETS", "PATHS")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8888AA")).Render(progHeader))
		lines = append(lines, mutedStyle.Render("  -------------------------------------------------------------"))

		for _, p := range data.Programs {
			pathsStr := strings.Join(p.Paths, ", ")
			if pathsStr == "" {
				pathsStr = mutedStyle.Render("(no target for current OS)")
			}
			targetInfo := fmt.Sprintf("%d target(s)", p.TargetCount)
			row := fmt.Sprintf("  %-16s %-14s %s", p.Name, targetInfo, pathsStr)
			lines = append(lines, valStyle.Render(row))
		}
		dotfilesContent = strings.Join(lines, "\n")
	}
	dotfilesCard := dashCardStyle.Width(contentWidth).Render(dotfilesContent)

	// 3. Quick Actions Section
	actionsHeader := dashHeaderStyle.Render("Quick Actions")
	actions := []string{
		fmt.Sprintf("%s %s", keyStyle.Render("[s]"), descStyle.Render("Save & Push local dotfiles to repository")),
		fmt.Sprintf("%s %s", keyStyle.Render("[y]"), descStyle.Render("Sync & Pull repository dotfiles to local system")),
		fmt.Sprintf("%s %s", keyStyle.Render("[r]"), descStyle.Render("Refresh repository git status")),
		fmt.Sprintf("%s %s", keyStyle.Render("[2]"), descStyle.Render("Manage Programs & Target Paths")),
		fmt.Sprintf("%s %s", keyStyle.Render("[3]"), descStyle.Render("Configure Git Remote & Cache Path")),
	}
	actionsContent := strings.Join(append([]string{actionsHeader}, actions...), "\n")
	actionsCard := dashCardStyle.Width(contentWidth).Render(actionsContent)

	return lipgloss.JoinVertical(lipgloss.Left, repoCard, dotfilesCard, actionsCard)
}
