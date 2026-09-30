package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	commandfile "github.com/exr462/go-dark/command/file"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/docker"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/state"
	"github.com/exr462/go-dark/task"
	"github.com/exr462/go-dark/ui/component"
	componentaction "github.com/exr462/go-dark/ui/component/action"
	componentmessage "github.com/exr462/go-dark/ui/component/message"
	"github.com/exr462/go-dark/window"
)

var checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case lsp.FileLoadedMsg:
		m.ui.Editor.SetValue(msg.Content)
		return m, nil
	case task.PipelineTaskStartedMsg:
		m.ui.StatusMsg = fmt.Sprintf("🏗️  Building: %s...", msg)
		return m, nil

	case task.PipelineTaskFinishedMsg:
		if msg.Err != nil {
			m.ui.StatusMsg = fmt.Sprintf("❌ Build Error on component: %s", msg.ProjectName)
		} else {
			m.ui.StatusMsg = fmt.Sprintf("✅ Component complete: %s", msg.ProjectName)
		}
		return m, nil

	case task.PipelineCompleteMsg:
		if msg.Success {
			m.ui.StatusMsg = "🎉 All workspace modules built successfully!"
		} else {
			m.ui.StatusMsg = "❌ Pipeline compilation aborted due to build errors."
		}
		m.ui.ViewState = model.StateDashboard
		return m, initializer.InitializeWorkspace(m.ui).OnAction()

	case config.GitStatusLoadedMsg:
		return m, componentaction.GitStatusLoaded(m.ui, msg)

	case componentmessage.WorkspaceRefreshedMsg:
		return m, componentaction.WorkspaceRefreshed(m.ui, msg)

	case config.GitStatusErrorMsg:
		m.ui.GitStatusOutput = fmt.Sprintf("❌ Error: %v", msg)
		return m, nil

	case config.GitBranchesLoadedMsg:
		m.ui.AvailableBranches = msg
		m.ui.SelectedGitBranch = 0
		return m, nil

	case config.GitBranchesErrorMsg:
		m.ui.AvailableBranches = []string{"main"}
		m.ui.SelectedGitBranch = 0
		m.ui.StatusMsg = fmt.Sprintf("❌ Git: %v", msg)
		return m, nil

	case config.GitCheckoutCompleteMsg:
		return m, componentaction.GitCheckoutComplete(m.ui, msg)

	case docker.DockerTelemetryMsg:
		m.ui.DockerTelemetry = docker.DockerStats(msg)
		return m, initializer.InitializeDockerTelemetry(m.ui).OnAction()

	case docker.DockerContainersMsg:
		m.ui.DockerContainers = msg
		return m, nil

	case tea.WindowSizeMsg:
		return m, window.Size(m.ui, msg)

	case model.FileLoadMsg:
		cmds = commandfile.FileLoad(m.ui, cmds)

	case model.ConfigRefreshedMsg:
		m.ui.Config = config.Config(msg)
		cmds = append(cmds, initializer.InitializeWorkspace(m.ui).OnAction())

	case state.StatusMsg:
		m.ui.StatusMsg = string(msg)
		if strings.Contains(m.ui.StatusMsg, "successfully completed") {
			for i := range m.ui.Config.Projects {
				gitDir := filepath.Join(m.ui.Config.Projects[i].Path, ".git")
				if _, err := os.Stat(gitDir); err == nil {
					m.ui.Config.Projects[i].Fetched = true
				}
			}
			_ = config.SaveConfig(m.ui.Config)
			return m, initializer.InitializeWorkspace(m.ui).OnAction()
		}
		return m, nil

	case componentmessage.BuildLogLineMsg:
		return m, componentaction.BuildLogLine(m.ui, msg)

	case componentmessage.BuildCompleteMsg:
		return m, componentaction.BuildComplete(m.ui, msg)

	case component.EditFileMsg:
		m.ui.ViewState = model.StateDashboard
		if msg.Err != nil {
			m.ui.LastError = msg.Err
			m.ui.StatusMsg = fmt.Sprintf("❌ Editor: %v", msg.Err)
			return m, nil
		}
		m.ui.ActiveCodeBuffer = msg.Content
		return m, nil

	case preflightMsg:
		log.Printf("preflight: %s index: %d", msg, m.ui.Index)
		// 1. Run the payload for the current index step if safe
		var loadCmd tea.Cmd
		if msg.Function != nil {
			loadCmd = msg.Function()
		}
		// 2. Advance to the next precheck index
		m.ui.Index++

		// 3. CRITICAL CHECK: Intercept failures or first-run blocks immediately
		//    Replace 'msg.Err != nil' with whatever field your preflightMsg carries for errors
		if m.ui.IsFirstRun {
			m.ui.IsFirstRun = false
			// Halt the process by shifting the active layout screen state
			m.ui.ViewState = model.StateGitConfigurationModal

			// Return immediately with optional clean visual logs.
			// Do NOT append m.preflight() here. The loop stops completely!
			return m, tea.Printf("%s [ HALT ] Setup interrupted. Redirection to Git Configuration...", checkMark)
		}

		// 4. Check if EVERYTHING is completed successfully
		if m.ui.Index >= len(m.ui.Prechecks) {
			m.ui.Done = true
			progressCmd := m.ui.Progress.SetPercent(1.0)

			return m, tea.Sequence(
				loadCmd,
				progressCmd,
				tea.Printf("%s [ OK ] Go-Dark Subsystems Primed.", checkMark),
				func() tea.Msg { return state.PreflightCompleteMsg{} },
			)
		}

		// 5. Continue processing remaining tasks smoothly
		nextTickCmd := preflight(m.ui)
		progressCmd := m.ui.Progress.SetPercent(float64(m.ui.Index) / float64(len(m.ui.Prechecks)))

		return m, tea.Batch(
			loadCmd,
			nextTickCmd,
			progressCmd,
		)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.ui.Spinner, cmd = m.ui.Spinner.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		updatedModel, cmd := m.ui.Progress.Update(msg)

		// Explicitly cast tea.Model back to progress.Model
		if pModel, ok := updatedModel.(progress.Model); ok {
			m.ui.Progress = pModel
		}
		return m, cmd

	case state.PreflightCompleteMsg:
		m.ui.ViewState = model.StateDashboard
		log.Printf("PreflightCompleteMsg: %v", m.ui.ViewState)
		return m, nil

	case tea.KeyMsg:
		// Global quit shortcuts
		if msg.String() == action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.QuitApplication) {
			return m, tea.Quit
		}

		// Route explicitly to active modal handler
		switch m.ui.ViewState {
		case model.StateHelpModal:
			return m, componentaction.HelpModal(m.ui, msg)
		case model.StateGitConfigurationModal:
			return m, componentaction.UpdateGitConfiguration(m.ui, msg)
		case model.StateGitOperationsModal:
			return m, componentaction.GitOpsModal(m.ui, msg)
		case model.StateJDKConfigModal:
			return m, componentaction.JDKModal(m.ui, msg)
		case model.StateMavenConfigModal:
			return m, componentaction.MvnModal(m.ui, msg)
		case model.StateBuildModal:
			return m, componentaction.BuildModal(m.ui, msg)
		case model.StateSessionLogsModal:
			return m, componentaction.SessionLogsModal(m.ui, msg)
		case model.StateFuzzyModal:
			return m, componentaction.FuzzyModal(m.ui, msg)
		case model.StateConfigDeckModal:
			return m, componentaction.ConfigDeckModal(m.ui, msg)
		case model.StateDockerModal:
			return m, componentaction.DockerModal(m.ui, msg)
		case model.StateShortcutConfigurationModal:
			return m, componentaction.ShortcutsConfigurationModal(m.ui, msg)
		case model.StateEditorModal:
			return m, componentaction.EditorModal(m.ui, msg)
		case model.StateDependencyConfigModal:
			// 🔍 SAFE VALIDATION INSIDE TARGET CONTEXT
			projIdx := m.ui.DepScreen.ActiveProjectIndex
			if projIdx < 0 || projIdx >= len(m.ui.Config.Projects) {
				m.ui.ViewState = model.StateDashboard
				return m, nil
			}
			return m, componentaction.DependencyScreen(m.ui, msg)
		case model.StateDashboard:
			fallthrough
		default:
			return m, componentaction.DashboardPortal(m.ui, m.providerFactory, msg)
		}
	}

	// Viewport event dispatching
	if m.ui.ViewState == model.StateDashboard {
		var viewCmd tea.Cmd
		m.ui.FileViewer, viewCmd = m.ui.FileViewer.Update(msg)
		cmds = append(cmds, viewCmd)
	}

	if m.ui.ViewState == model.StateFuzzyModal {
		var fuzzyCmd tea.Cmd
		m.ui.FuzzyViewer, fuzzyCmd = m.ui.FuzzyViewer.Update(msg)
		cmds = append(cmds, fuzzyCmd)
	}

	// Cleaned duplicate fallback block at the bottom
	return m, tea.Batch(cmds...)
}
