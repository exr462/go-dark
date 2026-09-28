package main

import (
	"log"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/state"
	"github.com/exr462/go-dark/task"
	"github.com/exr462/go-dark/ui/component"
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

	case component.WorkspaceRefreshedMsg:
		return m, component.WorkspaceRefreshed(m.ui, msg)

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

	case component.EditFileMsg:
		return m.editFile(msg)

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
			return m.updateHelpModal(msg)
		case model.StateGitConfigurationModal:
			return m, component.UpdateGitConfiguration(msg, m.ui)
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
