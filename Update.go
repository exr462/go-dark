package main

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/task"
	"github.com/exr462/go-dark/ui/components"
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
		return m.pipelineTaskStarted(msg)

	case task.PipelineTaskFinishedMsg:
		return m.pipelineTaskFinished(msg)

	case task.PipelineCompleteMsg:
		return m.pipelineComplete(msg)

	case config.GitStatusLoadedMsg:
		return m.gitStatusLoaded(msg)

	case config.WorkspaceRefreshedMsg:
		return m.workspaceRefreshed(msg)

	case config.GitStatusErrorMsg:
		return m.gitStatusError(msg)

	case config.GitBranchesLoadedMsg:
		return m.gitBranchesLoaded(msg)

	case config.GitBranchesErrorMsg:
		return m.gitBranchesError(msg)

	case config.GitCheckoutCompleteMsg:
		return m.gitCheckoutComplete(msg)

	case model.DockerTelemetryMsg:
		return m.dockerTelemetry(msg)

	case model.DockerContainersMsg:
		return m.dockerContainers(msg)

	case tea.WindowSizeMsg:
		return m.windowSize(msg)

	case model.FileLoadMsg:
		cmds = m.fileLoad(cmds)

	case model.ConfigRefreshedMsg:
		cmds = m.configRefreshed(msg, cmds)

	case model.StatusMsg:
		return m.status(msg)

	case model.BuildLogLineMsg:
		return m.buildLogLine(msg)

	case model.BuildCompleteMsg:
		return m.buildComplete(msg)

	case components.EditFileMsg:
		return m.editFile(msg)

	case preflightMsg:
		// 1. If we finished the last precheck, complete and quit
		if m.ui.Index >= len(m.ui.Prechecks)-1 {
			m.ui.Done = true
			m.ui.ViewState = model.StateDashboard
			return m, nil
		}

		// 2. Run the payload for the CURRENT index step first
		var loadCmd tea.Cmd
		if msg.Function != nil {
			loadCmd = msg.Function() // Execute precheck.Load
		}

		// 3. Advance to the next precheck index
		m.ui.Index++

		// 4. Queue up the NEXT precheck tick & update the progress bar
		nextTickCmd := m.preflight()
		progressCmd := m.ui.Progress.SetPercent(float64(m.ui.Index) / float64(len(m.ui.Prechecks)))

		return m, tea.Batch(
			loadCmd,
			nextTickCmd, // This wakes up the loop again for the next item!
			progressCmd,
			tea.Printf("%s Step completed", checkMark),
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

	case tea.KeyMsg:
		// Global quit shortcuts
		if msg.String() == action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.QuitApplication) {
			return m, tea.Quit
		}

		// Route explicitly to active modal handler
		switch m.ui.ViewState {
		case model.StateHelpModal:
			return m.updateHelpModal(msg)
		case model.StateGitConfigurationModal:
			return m.updateGitConfiguration(msg)
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
		case model.StateShortcutConfigurationModal:
			return m.updateShortcutsConfigurationModal(msg)
		case model.StateEditorModal:
			return m.updateEditorModal(msg)
		case model.StateDependencyConfigModal:
			// 🔍 SAFE VALIDATION INSIDE TARGET CONTEXT
			projIdx := m.ui.DepScreen.ActiveProjectIndex
			if projIdx < 0 || projIdx >= len(m.ui.Config.Projects) {
				m.ui.ViewState = model.StateDashboard
				return m, nil
			}
			return m.updateDependencyScreen(msg)
		case model.StateDashboard:
			fallthrough
		default:
			return m.updateDashboardPortal(msg)
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
