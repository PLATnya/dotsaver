package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// SettingsView provides an interactive configuration form for dotsaver settings.
type SettingsView struct {
	RemoteURLInput textinput.Model
	RepoPathInput  textinput.Model
	FocusIndex     int // 0 = RemoteURL, 1 = RepoPath
}

var (
	settingsCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#444455")).
				Padding(1, 2)

	settingsHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00D2FF")).
				MarginBottom(1)

	fieldLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EEEEEE"))

	fieldHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777"))

	settingsKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7D56F4"))

	settingsDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CCCCCC"))
)

// NewSettingsView initializes the settings form with the given remote URL and cache repo path.
func NewSettingsView(remoteURL, repoPath string) *SettingsView {
	urlInput := textinput.New()
	urlInput.Placeholder = "git@github.com:username/dotfiles.git (or https://...)"
	urlInput.SetValue(remoteURL)
	urlInput.CharLimit = 256
	urlInput.Width = 60
	urlInput.Focus()

	pathInput := textinput.New()
	pathInput.Placeholder = "Local path where git repository is cached"
	pathInput.SetValue(repoPath)
	pathInput.CharLimit = 256
	pathInput.Width = 60

	return &SettingsView{
		RemoteURLInput: urlInput,
		RepoPathInput:  pathInput,
		FocusIndex:     0,
	}
}

// SetValues updates the form values.
func (sv *SettingsView) SetValues(remoteURL, repoPath string) {
	sv.RemoteURLInput.SetValue(remoteURL)
	sv.RepoPathInput.SetValue(repoPath)
}

// NextFocus moves focus to the next input in the form.
func (sv *SettingsView) NextFocus() {
	sv.FocusIndex = (sv.FocusIndex + 1) % 2
	sv.updateFocus()
}

// PrevFocus moves focus to the previous input in the form.
func (sv *SettingsView) PrevFocus() {
	sv.FocusIndex = (sv.FocusIndex + 1) % 2
	sv.updateFocus()
}

func (sv *SettingsView) updateFocus() {
	if sv.FocusIndex == 0 {
		sv.RemoteURLInput.Focus()
		sv.RepoPathInput.Blur()
	} else {
		sv.RemoteURLInput.Blur()
		sv.RepoPathInput.Focus()
	}
}

// Render renders the settings form.
func (sv *SettingsView) Render(width int) string {
	contentWidth := width - 4
	if contentWidth < 50 {
		contentWidth = 76
	}

	formLines := []string{
		settingsHeaderStyle.Render("dotsaver Configuration"),
		"",
		fieldLabelStyle.Render("Remote Git Repository URL:"),
		fieldHelpStyle.Render("  Enter the Git clone URL for your dotfiles repository (SSH or HTTPS)."),
		fieldHelpStyle.Render("  Leave empty for local-only repository tracking."),
		sv.RemoteURLInput.View(),
		"",
		fieldLabelStyle.Render("Repository Cache Path:"),
		fieldHelpStyle.Render("  Local filesystem path where the repository clone is stored."),
		sv.RepoPathInput.View(),
		"",
		"",
		strings.Join([]string{
			settingsKeyStyle.Render("[Tab/↑/↓]"),
			settingsDescStyle.Render("Navigate Fields"),
			" • ",
			settingsKeyStyle.Render("[Enter]"),
			settingsDescStyle.Render("Save Settings & Verify Repo"),
			" • ",
			settingsKeyStyle.Render("[Esc]"),
			settingsDescStyle.Render("Cancel / Return to Dashboard"),
		}, " "),
	}

	return settingsCardStyle.Width(contentWidth).Render(strings.Join(formLines, "\n"))
}
