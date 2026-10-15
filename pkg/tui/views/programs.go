package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/dotsaver/pkg/manifest"
)

// ProgramsView manages the selection, navigation, and dialogs in the Programs tab.
type ProgramsView struct {
	SelectedProgIndex   int
	SelectedTargetIndex int
	FocusPane           int // 0 = Programs list, 1 = Targets list

	// Add Program Dialog State
	AddingProgram bool
	ProgNameInput textinput.Model

	// Add Target Dialog State
	AddingTarget     bool
	TargetProgName   string
	TargetOSInput    textinput.Model
	TargetPathInput  textinput.Model
	TargetAliasInput textinput.Model
	TargetFocusIndex int // 0 = OS, 1 = Path, 2 = Alias
}

var (
	progCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444455")).
			Padding(1, 2)

	progFocusedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(1, 2)

	dialogCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00D2FF")).
			Padding(1, 2)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D2FF")).
			MarginBottom(1)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777"))

	tagStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#A8DADC")).
			Padding(0, 1)
)

// NewProgramsView instantiates a fresh ProgramsView with initialized inputs.
func NewProgramsView() *ProgramsView {
	pi := textinput.New()
	pi.Placeholder = "e.g. nvim, git, zsh"
	pi.CharLimit = 64

	osInput := textinput.New()
	osInput.Placeholder = "windows or linux"
	osInput.CharLimit = 16

	pathInput := textinput.New()
	pathInput.Placeholder = "e.g. ~/.config/nvim or %APPDATA%/nvim"
	pathInput.CharLimit = 256

	aliasInput := textinput.New()
	aliasInput.Placeholder = "optional (defaults to filename)"
	aliasInput.CharLimit = 64

	return &ProgramsView{
		ProgNameInput:    pi,
		TargetOSInput:    osInput,
		TargetPathInput:  pathInput,
		TargetAliasInput: aliasInput,
	}
}

// StartAddProgram opens the Add Program text dialog.
func (pv *ProgramsView) StartAddProgram() {
	pv.AddingProgram = true
	pv.AddingTarget = false
	pv.ProgNameInput.Reset()
	pv.ProgNameInput.Focus()
}

// StartAddTarget opens the Add Target dialog for the selected program.
func (pv *ProgramsView) StartAddTarget(currentOS, progName string) {
	if progName == "" {
		return
	}
	pv.AddingTarget = true
	pv.AddingProgram = false
	pv.TargetProgName = progName

	pv.TargetOSInput.Reset()
	pv.TargetOSInput.SetValue(strings.ToLower(currentOS))
	pv.TargetPathInput.Reset()
	pv.TargetAliasInput.Reset()

	pv.TargetFocusIndex = 1 // Focus path first as OS is prefilled
	pv.TargetOSInput.Blur()
	pv.TargetPathInput.Focus()
	pv.TargetAliasInput.Blur()
}

// CancelDialog cancels and closes any open dialog.
func (pv *ProgramsView) CancelDialog() {
	pv.AddingProgram = false
	pv.AddingTarget = false
	pv.ProgNameInput.Blur()
	pv.TargetOSInput.Blur()
	pv.TargetPathInput.Blur()
	pv.TargetAliasInput.Blur()
}

// NextTargetField advances focus to the next field in the Add Target form.
func (pv *ProgramsView) NextTargetField() {
	pv.TargetFocusIndex = (pv.TargetFocusIndex + 1) % 3
	pv.updateTargetFieldFocus()
}

// PrevTargetField moves focus to the previous field in the Add Target form.
func (pv *ProgramsView) PrevTargetField() {
	pv.TargetFocusIndex = (pv.TargetFocusIndex + 2) % 3
	pv.updateTargetFieldFocus()
}

func (pv *ProgramsView) updateTargetFieldFocus() {
	pv.TargetOSInput.Blur()
	pv.TargetPathInput.Blur()
	pv.TargetAliasInput.Blur()

	switch pv.TargetFocusIndex {
	case 0:
		pv.TargetOSInput.Focus()
	case 1:
		pv.TargetPathInput.Focus()
	case 2:
		pv.TargetAliasInput.Focus()
	}
}

// TogglePane toggles focus between the Programs list and the Targets list.
func (pv *ProgramsView) TogglePane() {
	if pv.FocusPane == 0 {
		pv.FocusPane = 1
	} else {
		pv.FocusPane = 0
	}
}

// MoveUp moves the cursor up in the active pane.
func (pv *ProgramsView) MoveUp(m *manifest.Manifest) {
	if pv.FocusPane == 0 {
		if pv.SelectedProgIndex > 0 {
			pv.SelectedProgIndex--
			pv.SelectedTargetIndex = 0
		}
	} else {
		if pv.SelectedTargetIndex > 0 {
			pv.SelectedTargetIndex--
		}
	}
}

// MoveDown moves the cursor down in the active pane.
func (pv *ProgramsView) MoveDown(m *manifest.Manifest) {
	if m == nil {
		return
	}
	if pv.FocusPane == 0 {
		if pv.SelectedProgIndex < len(m.Programs)-1 {
			pv.SelectedProgIndex++
			pv.SelectedTargetIndex = 0
		}
	} else {
		prog := pv.SelectedProgram(m)
		if prog != nil && pv.SelectedTargetIndex < len(prog.Targets)-1 {
			pv.SelectedTargetIndex++
		}
	}
}

// SelectedProgram returns a pointer to the currently selected Program, or nil.
func (pv *ProgramsView) SelectedProgram(m *manifest.Manifest) *manifest.Program {
	if m == nil || len(m.Programs) == 0 {
		return nil
	}
	if pv.SelectedProgIndex < 0 {
		pv.SelectedProgIndex = 0
	}
	if pv.SelectedProgIndex >= len(m.Programs) {
		pv.SelectedProgIndex = len(m.Programs) - 1
	}
	return &m.Programs[pv.SelectedProgIndex]
}

// SelectedTarget returns the program name and selected target, or empty if none.
func (pv *ProgramsView) SelectedTarget(m *manifest.Manifest) (string, *manifest.Target) {
	prog := pv.SelectedProgram(m)
	if prog == nil || len(prog.Targets) == 0 {
		return "", nil
	}
	if pv.SelectedTargetIndex < 0 {
		pv.SelectedTargetIndex = 0
	}
	if pv.SelectedTargetIndex >= len(prog.Targets) {
		pv.SelectedTargetIndex = len(prog.Targets) - 1
	}
	return prog.Name, &prog.Targets[pv.SelectedTargetIndex]
}

// Render renders the Programs view or active dialogs.
func (pv *ProgramsView) Render(m *manifest.Manifest, currentOS string, width int) string {
	contentWidth := width - 4
	if contentWidth < 50 {
		contentWidth = 76
	}

	// 1. If Add Program dialog is active
	if pv.AddingProgram {
		return pv.renderAddProgramDialog(contentWidth)
	}

	// 2. If Add Target dialog is active
	if pv.AddingTarget {
		return pv.renderAddTargetDialog(contentWidth)
	}

	// 3. Normal View: Program List & Target Inspector
	leftWidth := (contentWidth / 3)
	if leftWidth < 24 {
		leftWidth = 24
	}
	rightWidth := contentWidth - leftWidth - 4
	if rightWidth < 30 {
		rightWidth = 30
	}

	leftStyle := progCardStyle
	rightStyle := progCardStyle
	if pv.FocusPane == 0 {
		leftStyle = progFocusedCardStyle
	} else {
		rightStyle = progFocusedCardStyle
	}

	// Render Left Pane: Programs
	var progLines []string
	progLines = append(progLines, headerStyle.Render("Programs"))
	if m == nil || len(m.Programs) == 0 {
		progLines = append(progLines, dimStyle.Render("No programs."))
		progLines = append(progLines, dimStyle.Render("Press 'a' to add."))
	} else {
		for i, prog := range m.Programs {
			line := fmt.Sprintf("%-14s (%d)", prog.Name, len(prog.Targets))
			if i == pv.SelectedProgIndex {
				if pv.FocusPane == 0 {
					progLines = append(progLines, selectedItemStyle.Render("▶ "+line))
				} else {
					progLines = append(progLines, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render("▶ "+line))
				}
			} else {
				progLines = append(progLines, normalItemStyle.Render("  "+line))
			}
		}
	}
	leftPane := leftStyle.Width(leftWidth).Render(strings.Join(progLines, "\n"))

	// Render Right Pane: Target Inspector
	var targetLines []string
	prog := pv.SelectedProgram(m)
	if prog == nil {
		targetLines = append(targetLines, headerStyle.Render("Targets"))
		targetLines = append(targetLines, dimStyle.Render("Select or add a program."))
	} else {
		targetLines = append(targetLines, headerStyle.Render(fmt.Sprintf("Targets for '%s'", prog.Name)))
		if len(prog.Targets) == 0 {
			targetLines = append(targetLines, dimStyle.Render("No target paths configured."))
			targetLines = append(targetLines, dimStyle.Render("Press 't' to add a target path."))
		} else {
			for i, t := range prog.Targets {
				osTag := fmt.Sprintf("[%s]", t.OS)
				aliasInfo := ""
				if t.Alias != "" {
					aliasInfo = fmt.Sprintf(" (alias: %s)", t.Alias)
				}
				row := fmt.Sprintf("%-9s %s%s", osTag, t.Path, aliasInfo)
				if i == pv.SelectedTargetIndex && pv.FocusPane == 1 {
					targetLines = append(targetLines, selectedItemStyle.Render("▶ "+row))
				} else {
					targetLines = append(targetLines, normalItemStyle.Render("  "+row))
				}
			}
		}
	}
	rightPane := rightStyle.Width(rightWidth).Render(strings.Join(targetLines, "\n"))

	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "  ", rightPane)

	// Key hints for Programs view
	hints := dimStyle.Render("[a] Add Program  •  [t] Add Target  •  [d] Delete  •  [Tab] Switch Pane  •  [↑/↓] Navigate")

	return lipgloss.JoinVertical(lipgloss.Left, panes, "\n"+hints)
}

func (pv *ProgramsView) renderAddProgramDialog(width int) string {
	dialogWidth := 50
	if dialogWidth > width {
		dialogWidth = width
	}

	body := strings.Join([]string{
		headerStyle.Render("Add New Program"),
		"Enter application or config name:",
		"",
		pv.ProgNameInput.View(),
		"",
		dimStyle.Render("[Enter] Confirm  •  [Esc] Cancel"),
	}, "\n")

	return dialogCardStyle.Width(dialogWidth).Render(body)
}

func (pv *ProgramsView) renderAddTargetDialog(width int) string {
	dialogWidth := 60
	if dialogWidth > width {
		dialogWidth = width
	}

	body := strings.Join([]string{
		headerStyle.Render(fmt.Sprintf("Add Target to '%s'", pv.TargetProgName)),
		"Target Operating System (windows or linux):",
		pv.TargetOSInput.View(),
		"",
		"Target Path (supports ~ and environment variables):",
		pv.TargetPathInput.View(),
		"",
		"Target Alias (optional, leave blank for default):",
		pv.TargetAliasInput.View(),
		"",
		dimStyle.Render("[Tab] Next Field  •  [Enter] Add Target  •  [Esc] Cancel"),
	}, "\n")

	return dialogCardStyle.Width(dialogWidth).Render(body)
}
