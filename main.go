package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
	"github.com/exr462/go-dark/ui/panels"
)

var factory = lsp.NewProviderFactory([]lsp.LanguageProvider{
	lsp.GoProvider{},
	lsp.KotlinProvider{},
	lsp.JavaProvider{},
})

var uiState = &model.UIState{
	InstallerStep:   model.StepSetGlobalPrefs,
	ActiveFocus:     model.FocusProjects,
	FileViewer:      viewport.New(30, 20),
	GitCommands:     []string{"checkout", "clone", "pull", "fetch", "status", "reset"},
	BuildOptions:    []string{"clean", "test", "compile", "package", "without tests", "full"},
	BuildLogs:       []string{"Console ready. Select option step to launch..."},
	Sessions:        make(map[int]*model.BuildSession),
	FuzzyQueryInput: textinput.New(),
	FuzzyViewer:     viewport.New(30, 20),
}

type appModel struct {
	state *model.UIState
}

func (m *appModel) Init() tea.Cmd {
	if m.state.ViewState == model.StateInstaller {
		return textinput.Blink
	}
	return tea.Batch(m.updateWorkspaceFiles(), m.pollDockerTelemetryCmd())
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case GitStatusLoadedMsg:
		m.state.GitStatusOutput = string(msg)
		if m.state.GitStatusOutput == "" {
			m.state.GitStatusOutput = "✨ Working tree clean."
		}
		return m, nil

	case WorkspaceRefreshedMsg:
		m.state.Files = msg.Files
		if m.state.SelectedProject >= 0 && m.state.SelectedProject < len(m.state.Config.Projects) {
			proj := m.state.Config.Projects[m.state.SelectedProject]
			m.state.TreeNodes = []model.FileNode{}
			if _, err := os.Stat(proj.Path); err == nil {
				m.buildTreeNodes(proj.Path, 0)
			}
		}
		return m, func() tea.Msg { return model.FileLoadMsg("sync") }

	case GitStatusErrorMsg:
		m.state.GitStatusOutput = fmt.Sprintf("❌ Error: %v", msg)
		return m, nil

	case GitBranchesLoadedMsg:
		m.state.AvailableBranches = msg
		m.state.SelectedGitBranch = 0
		return m, nil

	case GitBranchesErrorMsg:
		m.state.AvailableBranches = []string{"main"}
		m.state.SelectedGitBranch = 0
		m.state.StatusMsg = fmt.Sprintf("❌ Git: %v", msg)
		return m, nil

	case GitCheckoutCompleteMsg:
		if msg.Err != nil {
			m.state.StatusMsg = fmt.Sprintf("❌ %v", msg.Err)
		} else {
			m.state.StatusMsg = fmt.Sprintf("✅ %s", strings.TrimSpace(msg.Output))
			// Refresh fetched status across projects
			for i := range m.state.Config.Projects {
				gitDir := filepath.Join(m.state.Config.Projects[i].Path, ".git")
				if _, err := os.Stat(gitDir); err == nil {
					m.state.Config.Projects[i].Fetched = true
				}
			}
			_ = config.SaveConfig(m.state.Config)
		}
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()

	case model.DockerTelemetryMsg:
		m.state.DockerTelemetry = model.DockerStats(msg)
		return m, m.pollDockerTelemetryCmd()

	case model.DockerContainersMsg:
		m.state.DockerContainers = []model.DockerContainer(msg)
		return m, nil

	case tea.WindowSizeMsg:
		m.state.WindowWidth = msg.Width
		m.state.WindowHeight = msg.Height
		m.state.FileViewer.Width = (msg.Width / 2) - 4
		m.state.FileViewer.Height = max(msg.Height-8, 5)
		m.state.FuzzyViewer.Width = (msg.Width / 2) - 4
		m.state.FuzzyViewer.Height = max(msg.Height-12, 5)
		return m, nil

	case model.FileLoadMsg:
		if len(m.state.TreeNodes) > 0 {
			if m.state.SelectedFile >= len(m.state.TreeNodes) {
				m.state.SelectedFile = 0
			}
			cmds = append(cmds, m.readFileContentCmd())
		} else {
			m.state.FileViewer.SetContent("Empty or uncloned project repository.")
		}

	case model.ConfigRefreshedMsg:
		m.state.Config = config.Config(msg)
		cmds = append(cmds, m.updateWorkspaceFiles())

	case model.StatusMsg:
		m.state.StatusMsg = string(msg)
		if strings.Contains(m.state.StatusMsg, "successfully completed") {
			for i := range m.state.Config.Projects {
				gitDir := filepath.Join(m.state.Config.Projects[i].Path, ".git")
				if _, err := os.Stat(gitDir); err == nil {
					m.state.Config.Projects[i].Fetched = true
				}
			}
			_ = config.SaveConfig(m.state.Config)
			return m, m.updateWorkspaceFiles()
		}
		return m, nil

	case model.BuildLogLineMsg:
		if sess, exists := m.state.Sessions[msg.SessionID]; exists {
			if msg.Line != "" {
				line := msg.Line
				if regexp.MustCompile(`(?i)\[error\]|fail`).MatchString(line) {
					line = "\x1b[31;1m" + line + "\x1b[0m"
				} else if regexp.MustCompile(`(?i)\[warn`).MatchString(line) {
					line = "\x1b[33;1m" + line + "\x1b[0m"
				} else if regexp.MustCompile(`(?i)\[info\]|success`).MatchString(line) {
					line = "\x1b[32m" + line + "\x1b[0m"
				}

				sess.Logs = append(sess.Logs, line)
				if m.state.ViewState == model.StateBuildModal && m.state.ActiveSessionID == msg.SessionID {
					m.state.BuildLogs = sess.Logs
				}
			}
		}
		return m, func() tea.Msg {
			activeCh, exists := sessionChannels[msg.SessionID]
			if !exists {
				return nil
			}
			line, ok := <-activeCh
			if !ok {
				return model.BuildCompleteMsg{SessionID: msg.SessionID, Err: nil}
			}
			return model.BuildLogLineMsg{SessionID: msg.SessionID, Line: line}
		}

	case model.BuildCompleteMsg:
		if sess, exists := m.state.Sessions[msg.SessionID]; exists {
			sess.IsRunning = false
			sess.Logs = append(sess.Logs, "────────────────────────────────────────────────────────")
			if msg.Err != nil {
				sess.Logs = append(sess.Logs, fmt.Sprintf("❌ PROCESS TERMINATED WITH ERROR: %v", msg.Err))
			} else {
				sess.Logs = append(sess.Logs, "✅ PROCESS LOOP SUCCESSFULLY TERMINATED IN BACKGROUND.")
			}

			if m.state.ViewState == model.StateBuildModal && m.state.ActiveSessionID == msg.SessionID {
				m.state.BuildLogs = sess.Logs
				m.state.IsBuilding = false
			}
		}
		delete(sessionChannels, msg.SessionID)
		return m, nil

	case components.EditFileMsg:
		m.state.ViewState = model.StateDashboard
		if msg.Err != nil {
			m.state.LastError = msg.Err
			m.state.StatusMsg = fmt.Sprintf("❌ Editor: %v", msg.Err)
			return m, nil
		}
		m.state.ActiveCodeBuffer = msg.Content
		return m, nil

	case tea.KeyMsg:
		// Global quit shortcuts
		if msg.String() == "ctrl+c" || msg.String() == "ctrl+q" {
			return m, tea.Quit
		}

		// Route explicitly to active modal handler
		switch m.state.ViewState {
		case model.StateHelpModal:
			return m.updateHelpModal(msg)
		case model.StateInstaller:
			return m.updateInstaller(msg)
		case model.StateAddProjectModal:
			return m.updateModalForm(msg)
		case model.StateGitOperationsModal:
			return m.updateGitOpsModal(msg)
		case model.StateJDKConfigModal:
			return m.updateJDKModal(msg)
		case model.StateMavenConfigModal:
			return m.updateMvnModal(msg)
		case model.StateBuildModal:
			return m.updateBuildModal(msg)
		case model.StateSessionLogsModal:
			return m.updateSessionLogsModal(msg)
		case model.StateFuzzyModal:
			return m.updateFuzzyModal(msg)
		case model.StateConfigDeckModal:
			return m.updateConfigDeckModal(msg)
		case model.StateDockerModal:
			return m.updateDockerModal(msg)
		case model.StateEditorModal:
			return m, nil
		case model.StateDashboard:
			fallthrough
		default:
			return m.updateDashboardPortal(msg)
		}
	}

	// Viewport event dispatching
	if m.state.ViewState == model.StateDashboard {
		var viewCmd tea.Cmd
		m.state.FileViewer, viewCmd = m.state.FileViewer.Update(msg)
		cmds = append(cmds, viewCmd)
	}

	if m.state.ViewState == model.StateFuzzyModal {
		var fuzzyCmd tea.Cmd
		m.state.FuzzyViewer, fuzzyCmd = m.state.FuzzyViewer.Update(msg)
		cmds = append(cmds, fuzzyCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *appModel) View() string {
	switch m.state.ViewState {
	case model.StateHelpModal:
		return components.RenderHelpModal(m.state)
	case model.StateInstaller:
		return components.RenderInstaller(m.state)
	case model.StateAddProjectModal:
		return components.RenderModal(m.state)
	case model.StateGitOperationsModal:
		return components.RenderGitOpsModal(m.state)
	case model.StateJDKConfigModal:
		return components.RenderJDKConfigModal(m.state)
	case model.StateMavenConfigModal:
		return components.RenderMvnConfigModal(m.state)
	case model.StateBuildModal:
		return components.RenderBuildModal(m.state)
	case model.StateSessionLogsModal:
		return components.RenderSessionLogsModal(m.state)
	case model.StateFuzzyModal:
		return components.RenderFuzzyModal(m.state)
	case model.StateConfigDeckModal:
		return components.RenderConfigDeckModal(m.state)
	case model.StateDockerModal:
		return components.RenderDockerModal(m.state)
	case model.StateEditorModal:
		return "Opening external editor..."
	case model.StateDashboard:
		fallthrough
	default:
		return lipgloss.JoinVertical(
			lipgloss.Left,
			panels.RenderTopMenu(m.state),
			panels.RenderMainBody(m.state),
			panels.CreateFooter(m.state),
		)
	}
}

func main() {
	f, err := initLogger()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize log file: %v\n", err)
		os.Exit(1)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	cfg, isFirstRun := config.LoadConfig()
	_, gitErr := exec.LookPath("git")
	gitMissing := gitErr != nil
	home, _ := os.UserHomeDir()

	inputs := make([]textinput.Model, 11)
	for i := range inputs {
		inputs[i] = textinput.New()
	}

	inputs[0].Placeholder = "Global Workspace Base Path"
	inputs[0].SetValue(filepath.Join(home, "workspace"))
	inputs[1].Placeholder = "e.g. John Doe"
	inputs[2].Placeholder = "e.g. john@example.com"
	inputs[3].Placeholder = "Project Display Name"
	inputs[4].Placeholder = "folder-name"
	inputs[5].Placeholder = "java"
	inputs[6].Placeholder = "git@bitbucket.org:belgiantrain/repo.git"
	inputs[7].Placeholder = "Profile Name (e.g. Java-17)"
	inputs[8].Placeholder = "JAVA_HOME path (e.g. /usr/lib/jvm/...)"
	inputs[9].Placeholder = "Maven Profile Name (e.g. Maven-3.9)"
	inputs[10].Placeholder = "MAVEN_HOME directory path"

	initialState := model.StateDashboard
	if isFirstRun || gitMissing {
		initialState = model.StateInstaller
		if !gitMissing {
			inputs[0].Focus()
		}
	}

	uiState.Config = cfg
	uiState.ViewState = initialState
	uiState.Inputs = inputs
	uiState.GitMissing = gitMissing

	m := &appModel{
		state: uiState,
	}
	m.state.FuzzyQueryInput.Placeholder = "Type lookup phrase (e.g. controller)..."
	m.state.FuzzyQueryInput.CharLimit = 50

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}

func initLogger() (*os.File, error) {
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)
	log.Println("--- TUI Engine Session Started ---")
	return f, nil
}
