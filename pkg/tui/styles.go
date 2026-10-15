package tui

import "github.com/charmbracelet/lipgloss"

// Color Palette
var (
	PrimaryColor   = lipgloss.Color("#7D56F4") // Charm purple
	SecondaryColor = lipgloss.Color("#00D2FF") // Cyan accent
	SuccessColor   = lipgloss.Color("#04B575") // Green
	WarningColor   = lipgloss.Color("#FFB86C") // Amber / orange
	ErrorColor     = lipgloss.Color("#FF5555") // Red
	MutedColor     = lipgloss.Color("#666666") // Dim gray
	LightGrayColor = lipgloss.Color("#BBBBBB") // Soft text
	DarkBgColor    = lipgloss.Color("#1E1E2E") // Dark background
	PanelBgColor   = lipgloss.Color("#252538") // Subtle panel background
	BorderColor    = lipgloss.Color("#444455") // Subtle border
	FocusBorder    = lipgloss.Color("#7D56F4") // Active focus border
)

// App Title & Header Styles
var (
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(PrimaryColor).
			Padding(0, 1)

	AppSubtitleStyle = lipgloss.NewStyle().
				Foreground(LightGrayColor).
				PaddingLeft(1)

	HeaderContainer = lipgloss.NewStyle().
			MarginBottom(1)
)

// Tab Navigation Styles
var (
	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(PrimaryColor).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(LightGrayColor).
				Background(lipgloss.Color("#2A2A3C")).
				Padding(0, 2)

	TabGapStyle = lipgloss.NewStyle().
			Foreground(BorderColor).
			SetString(" ")

	TabContainerStyle = lipgloss.NewStyle().
				MarginBottom(1)
)

// Badges & Status Indicators
var (
	BadgeSuccessStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(SuccessColor).
				Padding(0, 1)

	BadgeWarningStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(WarningColor).
				Padding(0, 1)

	BadgeErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ErrorColor).
			Padding(0, 1)

	BadgeInfoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(SecondaryColor).
			Padding(0, 1)

	BadgeMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(MutedColor).
			Padding(0, 1)
)

// Card & Panel Styles
var (
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(1, 2)

	CardFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(FocusBorder).
				Padding(1, 2)

	SectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(SecondaryColor).
				MarginBottom(1)
)

// Status Notification Styles
var (
	StatusSuccessStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(ErrorColor).
				Bold(true)

	StatusInfoStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor)
)

// Help & Key Hint Styles
var (
	HelpContainerStyle = lipgloss.NewStyle().
				MarginTop(1)

	KeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor)

	KeyDescStyle = lipgloss.NewStyle().
			Foreground(LightGrayColor)

	KeySepStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			SetString(" • ")
)
