package tui

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/dotsaver/pkg/config"
	"github.com/user/dotsaver/pkg/git"
	"github.com/user/dotsaver/pkg/manifest"
	"github.com/user/dotsaver/pkg/syncer"
	"github.com/user/dotsaver/pkg/tui/views"
)

// Tab represents the active top-level tab.
type Tab int

const (
	TabDashboard Tab = iota
	TabPrograms
	TabSettings
)

// ModalType represents any currently active modal overlay.
type ModalType int

const (
	ModalNone ModalType = iota
	ModalBusy
	ModalConflict
)

// Typed messages for async tea.Cmd operations
type gitStatusMsg struct {
	status  string
	isClean bool
	branch  string
	err     error
}

type saveDoneMsg struct {
	report  *syncer.SaveReport
	pushOut string
	err     error
}

type syncDoneMsg struct {
	report   *syncer.SyncReport
	manifest *manifest.Manifest
	err      error
}

type syncConflictsMsg struct {
	conflicts []syncer.ConflictItem
	manifest  *manifest.Manifest
}

type saveSettingsDoneMsg struct {
	err error
}

// Model is the root Bubble Tea application model.
type Model struct {
	cfgPath   string
	cfg       *config.Config
	manifest  *manifest.Manifest
	gitSvc    *git.GitService
	syncer    *syncer.Syncer
	currentOS string

	activeTab   Tab
	activeModal ModalType

	// Spinner
	spinner     spinner.Model
	spinnerText string

	// Status & Notifications
	statusMsg   string
	statusIsErr bool
	gitStatus   string
	isGitClean  bool
	gitBranch   string

	// Conflicts
	conflicts []syncer.ConflictItem

	// Views
	programsView *views.ProgramsView
	settingsView *views.SettingsView

	// Window dimensions
	width  int
	height int
}

// NewModel instantiates and configures a new TUI Model.
func NewModel(cfgPath string) (*Model, error) {
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultConfigFilePath()
		if err != nil {
			return nil, fmt.Errorf("failed to determine default config path: %w", err)
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	gitSvc := git.New()
	syncEngine := syncer.NewSyncer(gitSvc)

	man, err := manifest.Load(cfg.RepoPath)
	if err != nil {
		man = manifest.New()
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	m := &Model{
		cfgPath:      cfgPath,
		cfg:          cfg,
		manifest:     man,
		gitSvc:       gitSvc,
		syncer:       syncEngine,
		currentOS:    runtime.GOOS,
		activeTab:    TabDashboard,
		activeModal:  ModalNone,
		spinner:      sp,
		gitBranch:    "main",
		gitStatus:    "Checking...",
		programsView: views.NewProgramsView(),
		settingsView: views.NewSettingsView(cfg.RemoteURL, cfg.RepoPath),
		width:        80,
		height:       24,
	}

	return m, nil
}

// Init begins background checks and spinner updates.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.refreshGitStatusCmd(),
	)
}

// Update processes Bubble Tea events, messages, and keybindings.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.activeModal == ModalBusy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case gitStatusMsg:
		m.gitStatus = msg.status
		m.isGitClean = msg.isClean
		m.gitBranch = msg.branch
		return m, nil

	case saveDoneMsg:
		m.activeModal = ModalNone
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Save failed: %v", msg.err)
			m.statusIsErr = true
		} else {
			savedCount := 0
			warnCount := 0
			if msg.report != nil {
				savedCount = len(msg.report.Saved)
				warnCount = len(msg.report.Warnings)
			}
			if warnCount > 0 {
				m.statusMsg = fmt.Sprintf("Saved %d targets with %d warning(s).", savedCount, warnCount)
			} else {
				m.statusMsg = fmt.Sprintf("Saved %d target(s) successfully.", savedCount)
			}
			m.statusIsErr = false
		}
		return m, m.refreshGitStatusCmd()

	case syncConflictsMsg:
		m.activeModal = ModalConflict
		m.conflicts = msg.conflicts
		if msg.manifest != nil {
			m.manifest = msg.manifest
		}
		return m, nil

	case syncDoneMsg:
		m.activeModal = ModalNone
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Sync failed: %v", msg.err)
			m.statusIsErr = true
		} else {
			if msg.manifest != nil {
				m.manifest = msg.manifest
			}
			updatedCount := 0
			backedUpCount := 0
			if msg.report != nil {
				updatedCount = len(msg.report.Updated)
				backedUpCount = len(msg.report.BackedUp)
			}
			m.statusMsg = fmt.Sprintf("Sync complete: %d restored, %d backed up (.bak).", updatedCount, backedUpCount)
			m.statusIsErr = false
		}
		return m, m.refreshGitStatusCmd()

	case saveSettingsDoneMsg:
		m.activeModal = ModalNone
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Settings saved, but repo setup had notice: %v", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = "Settings saved & repository cache initialized successfully."
			m.statusIsErr = false
			if loadedMan, err := manifest.Load(m.cfg.RepoPath); err == nil {
				m.manifest = loadedMan
			}
		}
		return m, m.refreshGitStatusCmd()

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global Quit
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// If busy modal is active, block key events
	if m.activeModal == ModalBusy {
		return m, nil
	}

	// If Conflict Modal is active
	if m.activeModal == ModalConflict {
		switch key {
		case "enter":
			return m, m.triggerSyncWithBackup()
		case "esc":
			m.activeModal = ModalNone
			m.statusMsg = "Sync skipped conflicting files."
			m.statusIsErr = false
			return m, nil
		default:
			return m, nil
		}
	}

	// Tab-specific key routing
	switch m.activeTab {
	case TabDashboard:
		return m.handleDashboardKey(key)
	case TabPrograms:
		return m.handleProgramsKey(msg)
	case TabSettings:
		return m.handleSettingsKey(msg)
	}

	return m, nil
}

func (m *Model) handleDashboardKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "1":
		m.activeTab = TabDashboard
	case "2":
		m.activeTab = TabPrograms
	case "3":
		m.activeTab = TabSettings
	case "tab":
		m.activeTab = TabPrograms
	case "s":
		return m, m.triggerSave()
	case "y":
		return m, m.triggerSync()
	case "r":
		m.statusMsg = "Refreshing git status..."
		m.statusIsErr = false
		return m, m.refreshGitStatusCmd()
	}
	return m, nil
}

func (m *Model) handleProgramsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// If Add Program dialog is active
	if m.programsView.AddingProgram {
		switch key {
		case "enter":
			progName := strings.TrimSpace(m.programsView.ProgNameInput.Value())
			if progName != "" {
				if _, err := m.manifest.AddProgram(progName); err != nil {
					m.statusMsg = fmt.Sprintf("Error: %v", err)
					m.statusIsErr = true
				} else {
					_ = m.manifest.Save(m.cfg.RepoPath)
					m.statusMsg = fmt.Sprintf("Added program '%s'", progName)
					m.statusIsErr = false
				}
			}
			m.programsView.CancelDialog()
			return m, nil
		case "esc":
			m.programsView.CancelDialog()
			return m, nil
		default:
			var cmd tea.Cmd
			m.programsView.ProgNameInput, cmd = m.programsView.ProgNameInput.Update(msg)
			return m, cmd
		}
	}

	// If Add Target dialog is active
	if m.programsView.AddingTarget {
		switch key {
		case "tab", "down":
			m.programsView.NextTargetField()
			return m, nil
		case "shift+tab", "up":
			m.programsView.PrevTargetField()
			return m, nil
		case "enter":
			osVal := strings.TrimSpace(m.programsView.TargetOSInput.Value())
			pathVal := strings.TrimSpace(m.programsView.TargetPathInput.Value())
			aliasVal := strings.TrimSpace(m.programsView.TargetAliasInput.Value())
			progName := m.programsView.TargetProgName

			if pathVal != "" {
				target := manifest.Target{
					OS:    osVal,
					Path:  pathVal,
					Alias: aliasVal,
				}
				if err := m.manifest.AddTarget(progName, target); err != nil {
					m.statusMsg = fmt.Sprintf("Error adding target: %v", err)
					m.statusIsErr = true
				} else {
					_ = m.manifest.Save(m.cfg.RepoPath)
					m.statusMsg = fmt.Sprintf("Added target to '%s'", progName)
					m.statusIsErr = false
				}
			}
			m.programsView.CancelDialog()
			return m, nil
		case "esc":
			m.programsView.CancelDialog()
			return m, nil
		default:
			var cmd tea.Cmd
			switch m.programsView.TargetFocusIndex {
			case 0:
				m.programsView.TargetOSInput, cmd = m.programsView.TargetOSInput.Update(msg)
			case 1:
				m.programsView.TargetPathInput, cmd = m.programsView.TargetPathInput.Update(msg)
			case 2:
				m.programsView.TargetAliasInput, cmd = m.programsView.TargetAliasInput.Update(msg)
			}
			return m, cmd
		}
	}

	// Normal Programs view keybindings
	switch key {
	case "q":
		return m, tea.Quit
	case "1":
		m.activeTab = TabDashboard
	case "2":
		m.activeTab = TabPrograms
	case "3":
		m.activeTab = TabSettings
	case "tab":
		m.programsView.TogglePane()
	case "up", "k":
		m.programsView.MoveUp(m.manifest)
	case "down", "j":
		m.programsView.MoveDown(m.manifest)
	case "a":
		m.programsView.StartAddProgram()
	case "t":
		prog := m.programsView.SelectedProgram(m.manifest)
		if prog != nil {
			m.programsView.StartAddTarget(m.currentOS, prog.Name)
		} else {
			m.statusMsg = "Add a program first before adding targets."
			m.statusIsErr = true
		}
	case "d":
		if m.programsView.FocusPane == 0 {
			prog := m.programsView.SelectedProgram(m.manifest)
			if prog != nil {
				progName := prog.Name
				m.manifest.RemoveProgram(progName)
				_ = m.manifest.Save(m.cfg.RepoPath)
				m.statusMsg = fmt.Sprintf("Deleted program '%s'", progName)
				m.statusIsErr = false
			}
		} else {
			progName, target := m.programsView.SelectedTarget(m.manifest)
			if target != nil {
				m.manifest.RemoveTarget(progName, target.OS, target.Path)
				_ = m.manifest.Save(m.cfg.RepoPath)
				m.statusMsg = fmt.Sprintf("Deleted target for '%s'", progName)
				m.statusIsErr = false
			}
		}
	case "s":
		return m, m.triggerSave()
	case "y":
		return m, m.triggerSync()
	}

	return m, nil
}

func (m *Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+1":
		m.activeTab = TabDashboard
		return m, nil
	case "ctrl+2":
		m.activeTab = TabPrograms
		return m, nil
	case "tab", "down":
		m.settingsView.NextFocus()
		return m, nil
	case "shift+tab", "up":
		m.settingsView.PrevFocus()
		return m, nil
	case "enter":
		return m, m.triggerSaveSettings()
	case "esc":
		m.activeTab = TabDashboard
		return m, nil
	default:
		var cmd tea.Cmd
		if m.settingsView.FocusIndex == 0 {
			m.settingsView.RemoteURLInput, cmd = m.settingsView.RemoteURLInput.Update(msg)
		} else {
			m.settingsView.RepoPathInput, cmd = m.settingsView.RepoPathInput.Update(msg)
		}
		return m, cmd
	}
}

func (m *Model) triggerSave() tea.Cmd {
	m.activeModal = ModalBusy
	m.spinnerText = "Saving dotfiles to repository cache & pushing..."
	return func() tea.Msg {
		_ = m.gitSvc.EnsureRepo(m.cfg.RemoteURL, m.cfg.RepoPath)
		rep, err := m.syncer.Save(m.cfg.RepoPath, m.manifest, m.currentOS)
		if err != nil {
			return saveDoneMsg{report: rep, err: err}
		}
		hostname, _ := os.Hostname()
		commitMsg := fmt.Sprintf("dotsaver: save on %s (%s)", hostname, m.currentOS)

		pushOut, pushErr := m.gitSvc.CommitAndPush(m.cfg.RepoPath, commitMsg)
		if pushErr != nil && m.cfg.RemoteURL == "" {
			// Local repository only, ignore remote push failure
			pushErr = nil
			pushOut = "Committed to local repository."
		}
		return saveDoneMsg{report: rep, pushOut: pushOut, err: pushErr}
	}
}

func (m *Model) triggerSync() tea.Cmd {
	m.activeModal = ModalBusy
	m.spinnerText = "Pulling from remote & checking dotfiles..."
	return func() tea.Msg {
		_ = m.gitSvc.EnsureRepo(m.cfg.RemoteURL, m.cfg.RepoPath)
		if m.cfg.RemoteURL != "" {
			_, _ = m.gitSvc.Pull(m.cfg.RepoPath)
		}
		loadedMan, err := manifest.Load(m.cfg.RepoPath)
		if err != nil {
			return syncDoneMsg{err: err}
		}

		conflicts, err := m.syncer.DetectConflicts(m.cfg.RepoPath, loadedMan, m.currentOS)
		if err != nil {
			return syncDoneMsg{err: err}
		}
		if len(conflicts) > 0 {
			return syncConflictsMsg{conflicts: conflicts, manifest: loadedMan}
		}

		rep, err := m.syncer.Sync(m.cfg.RepoPath, loadedMan, m.currentOS, false)
		return syncDoneMsg{report: rep, manifest: loadedMan, err: err}
	}
}

func (m *Model) triggerSyncWithBackup() tea.Cmd {
	m.activeModal = ModalBusy
	m.spinnerText = "Synchronizing dotfiles and backing up conflicts to .bak..."
	return func() tea.Msg {
		rep, err := m.syncer.Sync(m.cfg.RepoPath, m.manifest, m.currentOS, true)
		return syncDoneMsg{report: rep, manifest: m.manifest, err: err}
	}
}

func (m *Model) triggerSaveSettings() tea.Cmd {
	m.activeModal = ModalBusy
	m.spinnerText = "Saving configuration & verifying repository..."
	remoteURL := strings.TrimSpace(m.settingsView.RemoteURLInput.Value())
	repoPath := strings.TrimSpace(m.settingsView.RepoPathInput.Value())

	return func() tea.Msg {
		m.cfg.RemoteURL = remoteURL
		m.cfg.RepoPath = repoPath
		if err := m.cfg.Save(m.cfgPath); err != nil {
			return saveSettingsDoneMsg{err: fmt.Errorf("failed to save config: %w", err)}
		}

		err := m.gitSvc.EnsureRepo(m.cfg.RemoteURL, m.cfg.RepoPath)
		return saveSettingsDoneMsg{err: err}
	}
}

func (m *Model) refreshGitStatusCmd() tea.Cmd {
	return func() tea.Msg {
		if m.cfg == nil || m.cfg.RepoPath == "" {
			return gitStatusMsg{status: "Uninitialized", isClean: false, branch: "main"}
		}
		clean, err := m.gitSvc.IsClean(m.cfg.RepoPath)
		if err != nil {
			return gitStatusMsg{status: "Uninitialized", isClean: false, branch: "main", err: err}
		}
		statusText := "Changes Pending"
		if clean {
			statusText = "Clean"
		}
		return gitStatusMsg{
			status:  statusText,
			isClean: clean,
			branch:  "main",
		}
	}
}

// View renders the TUI layout.
func (m *Model) View() string {
	// If modal overlay active
	if m.activeModal == ModalBusy {
		return views.RenderBusyModal(m.spinnerText, m.spinner.View(), m.width, m.height)
	}
	if m.activeModal == ModalConflict {
		return views.RenderConflictModal(m.conflicts, m.width, m.height)
	}

	// 1. Header & Title Bar
	title := AppTitleStyle.Render(" dotsaver ")
	subtitle := AppSubtitleStyle.Render("cross-platform dotfile sync & backup")
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle)

	// 2. Tab Navigation
	var t1, t2, t3 string
	if m.activeTab == TabDashboard {
		t1 = TabActiveStyle.Render("1. Dashboard")
		t2 = TabInactiveStyle.Render("2. Programs")
		t3 = TabInactiveStyle.Render("3. Settings")
	} else if m.activeTab == TabPrograms {
		t1 = TabInactiveStyle.Render("1. Dashboard")
		t2 = TabActiveStyle.Render("2. Programs")
		t3 = TabInactiveStyle.Render("3. Settings")
	} else {
		t1 = TabInactiveStyle.Render("1. Dashboard")
		t2 = TabInactiveStyle.Render("2. Programs")
		t3 = TabActiveStyle.Render("3. Settings")
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, t1, TabGapStyle.Render(), t2, TabGapStyle.Render(), t3)

	// 3. Active Tab Body
	var body string
	switch m.activeTab {
	case TabDashboard:
		dashData := views.DashboardData{
			RemoteURL:     m.cfg.RemoteURL,
			RepoPath:      m.cfg.RepoPath,
			GitBranch:     m.gitBranch,
			GitStatus:     m.gitStatus,
			IsGitClean:    m.isGitClean,
			CurrentOS:     m.currentOS,
			TotalPrograms: len(m.manifest.Programs),
		}
		targetsByProg := m.manifest.TargetsForOS(m.currentOS)
		totalTargets := 0
		for _, prog := range m.manifest.Programs {
			var paths []string
			if tgts, ok := targetsByProg[prog.Name]; ok {
				for _, t := range tgts {
					paths = append(paths, t.Path)
					totalTargets++
				}
			}
			dashData.Programs = append(dashData.Programs, views.ProgramSummary{
				Name:        prog.Name,
				TargetCount: len(prog.Targets),
				Paths:       paths,
			})
		}
		dashData.TotalTargets = totalTargets
		body = views.RenderDashboard(dashData, m.width)

	case TabPrograms:
		body = m.programsView.Render(m.manifest, m.currentOS, m.width)

	case TabSettings:
		body = m.settingsView.Render(m.width)
	}

	// 4. Status Notification Bar
	var statusBar string
	if m.statusMsg != "" {
		if m.statusIsErr {
			statusBar = StatusErrorStyle.Render("● " + m.statusMsg)
		} else {
			statusBar = StatusSuccessStyle.Render("● " + m.statusMsg)
		}
	}

	// 5. Footer / Global Help Bar
	footer := strings.Join([]string{
		KeyStyle.Render("[1/2/3/Tab]"),
		KeyDescStyle.Render("Switch Tabs"),
		KeySepStyle.Render(),
		KeyStyle.Render("[s]"),
		KeyDescStyle.Render("Save"),
		KeySepStyle.Render(),
		KeyStyle.Render("[y]"),
		KeyDescStyle.Render("Sync"),
		KeySepStyle.Render(),
		KeyStyle.Render("[q]"),
		KeyDescStyle.Render("Quit"),
	}, " ")

	parts := []string{
		header,
		tabs,
		body,
	}
	if statusBar != "" {
		parts = append(parts, statusBar)
	}
	parts = append(parts, footer)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
