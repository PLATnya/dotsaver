package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// SettingsMode represents the interaction mode within the Settings tab.
type SettingsMode int

const (
	SettingsModeSelect SettingsMode = iota
	SettingsModeInput
)

// SettingsView provides an interactive configuration form for dotsaver settings.
type SettingsView struct {
	RemoteURLInput textinput.Model
	RepoPathInput  textinput.Model
	FocusIndex     int // 0 = RemoteURL, 1 = RepoPath
	Mode           SettingsMode
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

	selectModeBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#00D2FF")).
				Padding(0, 1)

	inputModeBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#00FF87")).
				Padding(0, 1)

	fieldLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EEEEEE"))

	selectedIndicatorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00D2FF"))

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
	urlInput.Blur()

	pathInput := textinput.New()
	pathInput.Placeholder = "Local path where git repository is cached"
	pathInput.SetValue(repoPath)
	pathInput.CharLimit = 256
	pathInput.Width = 60
	pathInput.Blur()

	return &SettingsView{
		RemoteURLInput: urlInput,
		RepoPathInput:  pathInput,
		FocusIndex:     0,
		Mode:           SettingsModeSelect,
	}
}

// SetValues updates the form values.
func (sv *SettingsView) SetValues(remoteURL, repoPath string) {
	sv.RemoteURLInput.SetValue(remoteURL)
	sv.RepoPathInput.SetValue(repoPath)
}

// SetMode switches between Select and Input modes and updates focus.
func (sv *SettingsView) SetMode(mode SettingsMode) {
	sv.Mode = mode
	sv.updateFocus()
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
	if sv.Mode == SettingsModeSelect {
		sv.RemoteURLInput.Blur()
		sv.RepoPathInput.Blur()
		return
	}

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

	var modeBadge string
	var footerHints string

	urlPrefix := "  "
	pathPrefix := "  "
	if sv.FocusIndex == 0 {
		urlPrefix = selectedIndicatorStyle.Render("> ")
	} else {
		pathPrefix = selectedIndicatorStyle.Render("> ")
	}

	if sv.Mode == SettingsModeSelect {
		modeBadge = selectModeBadgeStyle.Render("SELECT MODE")
		footerHints = strings.Join([]string{
			settingsKeyStyle.Render("[i]"),
			settingsDescStyle.Render("Edit Input"),
			" • ",
			settingsKeyStyle.Render("[↑/↓]"),
			settingsDescStyle.Render("Select Line"),
			" • ",
			settingsKeyStyle.Render("[1/2/3/Tab]"),
			settingsDescStyle.Render("Switch Tabs"),
			" • ",
			settingsKeyStyle.Render("[Enter]"),
			settingsDescStyle.Render("Save Settings"),
			" • ",
			settingsKeyStyle.Render("[q]"),
			settingsDescStyle.Render("Quit"),
		}, " ")
	} else {
		modeBadge = inputModeBadgeStyle.Render("INPUT MODE")
		footerHints = strings.Join([]string{
			settingsKeyStyle.Render("[Alt+i / Esc]"),
			settingsDescStyle.Render("Exit Input Mode"),
			" • ",
			settingsKeyStyle.Render("[Tab/↑/↓]"),
			settingsDescStyle.Render("Next Line"),
			" • ",
			settingsKeyStyle.Render("[Enter]"),
			settingsDescStyle.Render("Save Settings"),
		}, " ")
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center, settingsHeaderStyle.Render("dotsaver Configuration"), "  ", modeBadge)

	formLines := []string{
		header,
		"",
		urlPrefix + fieldLabelStyle.Render("Remote Git Repository URL:"),
		fieldHelpStyle.Render("    Enter the Git clone URL for your dotfiles repository (SSH or HTTPS)."),
		fieldHelpStyle.Render("    Leave empty for local-only repository tracking."),
		"  " + sv.RemoteURLInput.View(),
		"",
		pathPrefix + fieldLabelStyle.Render("Repository Cache Path:"),
		fieldHelpStyle.Render("    Local filesystem path where the repository clone is stored."),
		"  " + sv.RepoPathInput.View(),
		"",
		"",
		footerHints,
	}

	return settingsCardStyle.Width(contentWidth).Render(strings.Join(formLines, "\n"))
}
