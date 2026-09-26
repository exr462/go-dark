package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/task"
	"github.com/exr462/go-dark/ui/components"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
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

	case tea.KeyMsg:
		// Global quit shortcuts
		if msg.String() == "ctrl+c" || msg.String() == "ctrl+q" {
			return m, tea.Quit
		}

		// Route explicitly to active modal handler
		switch m.state.ViewState {
		case model.StateHelpModal:
			return m.updateHelpModal(msg)
		case model.StateGitConfigurationModal:
			return m.updateGitConfiguration(msg)
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
		case model.StateDependencyConfigModal:
			// 🔍 SAFE VALIDATION INSIDE TARGET CONTEXT
			projIdx := m.state.DepScreen.ActiveProjectIndex
			if projIdx < 0 || projIdx >= len(m.state.Config.Projects) {
				m.state.ViewState = model.StateDashboard
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

	// Cleaned duplicate fallback block at the bottom
	return m, tea.Batch(cmds...)
}
