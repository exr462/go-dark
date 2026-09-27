package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) prepareProfileScreen(viewState model.ApplicationViewState, name model.InputField, path model.InputField) (tea.Model, tea.Cmd) {
	if viewState == model.StateJDKConfigModal {
		m.state.JDKStep = model.StepSelectJDKAction
	} else {
		m.state.MavenStep = model.StepSelectMvnAction
	}
	m.state.ViewState = viewState
	m.state.Inputs[name].SetValue("")
	m.state.Inputs[path].SetValue("")
	m.state.SelectedMenuIndex = 0
	m.state.SelectedJDKIndex = 0
	return m, nil
}

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateDashboardPortal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.QuitApplication):
		return m, tea.Quit

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenGitOperations):
		if len(m.state.Config.Projects) > 0 {
			m.state.ViewState = model.StateGitOperationsModal
			m.state.GitOperationStep = model.StepSelectGitProject
			m.state.SelectedGitProject = m.state.SelectedProject
			m.state.SelectedGitCommand = 0
			return m, nil
		}
		return m, nil
	case "d": // 👈 Pressing 'd' over a highlighted row opens its dependency mapper
		if len(m.state.Config.Projects) > 0 && m.state.SelectedProject >= 0 {
			activeProj := m.state.Config.Projects[m.state.SelectedProject]

			// Initialize options excluding self to prevent cyclical dependencies
			m.state.DepScreen = model.NewDependencyScreen(activeProj.Name, m.state.Config.Projects)
			m.state.DepScreen.ActiveProjectIndex = m.state.SelectedProject

			// Flip view state boundary to render configuration modal
			m.state.ViewState = model.StateDependencyConfigModal
			return m, nil
		}
		// Add this case inside your updateDashboardPortal switch-case handler:
	case "P": // 👈 Pressing Shift+P triggers the parallel DAG build
		m.state.StatusMsg = "🏗️ Resolving dependency graph and starting parallel builds..."

		// Fire off the background compilation using your exact AvailableProjects layout
		// Limits the machine execution block to a safe ceiling of 4 concurrent threads
		return m, m.TriggerPipelineCmd(4)

	case "B": // 👈 Capital 'B' triggers the full parallel build pipeline
		m.state.StatusMsg = "🏗️ Initializing parallel build graph..."
		m.state.ViewState = model.StateBuildModal // Optional: switch to a loading/progress view

		// Pass your max concurrency limit (e.g., 4 simultaneous builds)
		return m, m.TriggerPipelineCmd(4)
	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenEditShortcuts):
		m.state.PreviousViewState = model.StateDashboard
		m.loadShortcutsOnInputs()
		m.state.ViewState = model.StateShortcutConfigurationModal
		return m, nil
	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenFuzzy):
		if len(m.state.Config.Projects) > 0 {
			m.state.ViewState = model.StateFuzzyModal
			m.state.FuzzyMode = model.FuzzyModeFiles
			m.state.FuzzyQueryInput.SetValue("")
			m.state.FuzzyResults = []model.FuzzyResult{}
			m.state.SelectedFuzzy = 0
			m.state.FuzzyQueryInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenDocker):
		m.state.ViewState = model.StateDockerModal
		m.state.SelectedDockerRow = 0
		return m, m.fetchDockerContainersCmd()

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenJdk):
		m.state.PreviousViewState = model.StateDashboard
		return m.prepareProfileScreen(model.StateJDKConfigModal, model.JdkName, model.JdkPath)

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenMvn):
		m.state.PreviousViewState = model.StateDashboard
		return m.prepareProfileScreen(model.StateMavenConfigModal, model.MvnName, model.MvnPath)

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenConfiguration):
		m.state.ViewState = model.StateConfigDeckModal
		m.state.SelectedConfigOption = 0
		return m, nil

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenBuild):
		if len(m.state.Config.Projects) > 0 {
			m.state.ViewState = model.StateBuildModal
			m.state.SelectedBuildOption = 0
			m.state.BuildLogs = []string{"Console engine ready. Select command step to initialize stream..."}
			m.state.IsBuilding = false
			return m, nil
		}
		return m, nil

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenSession):
		m.state.ViewState = model.StateSessionLogsModal
		m.state.ViewingSessionID = 0
		return m, nil

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenHelp):
		m.state.ViewState = model.StateHelpModal
		return m, nil

	case kbd.Tab:
		m.state.ActiveFocus = model.FocusArea((int(m.state.ActiveFocus) + 1) % 3)
		return m, nil

	case "up", "k":
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProject > 0 {
			m.state.SelectedProject--
			m.refreshRightPaneFromSelectedProject()
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile > 0 {
			m.state.SelectedFile--
			cmds = append(cmds, m.readFileContentCmd())
		}

	case "down", "j":
		if m.state.ActiveFocus == model.FocusProjects && m.state.SelectedProject < len(m.state.Config.Projects)-1 {
			m.state.SelectedProject++
			m.refreshRightPaneFromSelectedProject()
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.state.ActiveFocus == model.FocusTree && m.state.SelectedFile < len(m.state.TreeNodes)-1 {
			m.state.SelectedFile++
			cmds = append(cmds, m.readFileContentCmd())
		}

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save), "right", "l":
		if m.state.ActiveFocus == model.FocusMenu {
			cmds = append(cmds, m.executeActiveMenuAction())
		} else if m.state.ActiveFocus == model.FocusTree && len(m.state.TreeNodes) > 0 {
			idx := m.state.SelectedFile
			if idx >= 0 && idx < len(m.state.TreeNodes) {
				if m.state.TreeNodes[idx].IsDir {
					m.state.TreeNodes[idx].IsExpanded = !m.state.TreeNodes[idx].IsExpanded
					m.rebuildActiveTree()
				} else {
					cmds = append(cmds, m.readFileContentCmd())
				}
			}
		}

	case "left", "backspace":
		if m.state.ActiveFocus == model.FocusTree && len(m.state.TreeNodes) > 0 {
			idx := m.state.SelectedFile
			if idx >= 0 && idx < len(m.state.TreeNodes) {
				if m.state.TreeNodes[idx].IsDir && m.state.TreeNodes[idx].IsExpanded {
					m.state.TreeNodes[idx].IsExpanded = false
					m.rebuildActiveTree()
				}
			}
		}

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenEditFile):
		m.state.ViewState = model.StateEditorModal
		m.state.FocusedInput = model.EditContent
		return m, m.loadFileCmd()
	}

	return m, tea.Batch(cmds...)
}
