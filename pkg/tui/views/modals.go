package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/user/dotsaver/pkg/syncer"
)

var (
	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF5555")).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(1, 2)

	busyBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(1, 4)

	modalHeaderAlert = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF5555")).
				MarginBottom(1)

	modalHeaderBusy = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D2FF")).
			MarginBottom(1)

	conflictItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFB86C"))

	modalActionKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	modalActionDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EEEEEE"))

	modalHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

// RenderConflictModal renders a review dialog displaying all detected file conflicts.
func RenderConflictModal(conflicts []syncer.ConflictItem, width, height int) string {
	boxWidth := width - 10
	if boxWidth < 50 {
		boxWidth = 60
	}
	if boxWidth > 90 {
		boxWidth = 90
	}

	var lines []string
	lines = append(lines, modalHeaderAlert.Render("⚠ Sync Conflicts Detected"))
	lines = append(lines, "The following local configuration files differ from the repository version:")
	lines = append(lines, "")

	for _, c := range conflicts {
		row := fmt.Sprintf("  • [%s] %s", c.Program, c.TargetPath)
		lines = append(lines, conflictItemStyle.Render(row))
	}

	lines = append(lines, "")
	lines = append(lines, modalHelpStyle.Render("If you overwrite, an automatic backup (.bak) will be created first."))
	lines = append(lines, "")
	lines = append(lines, strings.Join([]string{
		modalActionKey.Render("[Enter]"),
		modalActionDesc.Render("Overwrite & Create Backup"),
		"    ",
		modalActionKey.Render("[Esc]"),
		modalActionDesc.Render("Skip Conflicted Files"),
	}, " "))

	box := modalBoxStyle.Width(boxWidth).Render(strings.Join(lines, "\n"))
	return centerModal(box, width, height)
}

// RenderBusyModal renders a clean overlay spinner with the provided operation description.
func RenderBusyModal(spinnerText string, spinnerView string, width, height int) string {
	boxWidth := 50
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	content := strings.Join([]string{
		modalHeaderBusy.Render("Please Wait"),
		"",
		fmt.Sprintf("  %s  %s", spinnerView, spinnerText),
		"",
	}, "\n")

	box := busyBoxStyle.Width(boxWidth).Render(content)
	return centerModal(box, width, height)
}

func centerModal(box string, width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
