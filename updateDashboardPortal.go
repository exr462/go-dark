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
		m.ui.JDKStep = model.StepSelectJDKAction
	} else {
		m.ui.MavenStep = model.StepSelectMvnAction
	}
	m.ui.ViewState = viewState
	m.ui.Inputs[name].SetValue("")
	m.ui.Inputs[path].SetValue("")
	m.ui.SelectedMenuIndex = 0
	m.ui.SelectedJDKIndex = 0
	return m, nil
}

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateDashboardPortal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.QuitApplication):
		return m, tea.Quit

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenGitOperations):
		if len(m.ui.Config.Projects) > 0 {
			m.ui.ViewState = model.StateGitOperationsModal
			m.ui.GitOperationStep = model.StepSelectGitProject
			m.ui.SelectedGitProject = m.ui.SelectedProject
			m.ui.SelectedGitCommand = 0
			return m, nil
		}
		return m, nil
	case "d": // 👈 Pressing 'd' over a highlighted row opens its dependency mapper
		if len(m.ui.Config.Projects) > 0 && m.ui.SelectedProject >= 0 {
			activeProj := m.ui.Config.Projects[m.ui.SelectedProject]

			// Initialize options excluding self to prevent cyclical dependencies
			m.ui.DepScreen = model.NewDependencyScreen(activeProj.Name, m.ui.Config.Projects)
			m.ui.DepScreen.ActiveProjectIndex = m.ui.SelectedProject

			// Flip view state boundary to render configuration modal
			m.ui.ViewState = model.StateDependencyConfigModal
			return m, nil
		}
		// Add this case inside your updateDashboardPortal switch-case handler:
	case "P": // 👈 Pressing Shift+P triggers the parallel DAG build
		m.ui.StatusMsg = "🏗️ Resolving dependency graph and starting parallel builds..."

		// Fire off the background compilation using your exact AvailableProjects layout
		// Limits the machine execution block to a safe ceiling of 4 concurrent threads
		return m, m.TriggerPipelineCmd()

	case "B": // 👈 Capital 'B' triggers the full parallel build pipeline
		m.ui.StatusMsg = "🏗️ Initializing parallel build graph..."
		m.ui.ViewState = model.StateBuildModal // Optional: switch to a loading/progress view

		// Pass your max concurrency limit (e.g., 4 simultaneous builds)
		return m, m.TriggerPipelineCmd()
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenEditShortcuts):
		m.ui.PreviousViewState = model.StateDashboard
		m.loadShortcutsOnInputs()
		m.ui.ViewState = model.StateShortcutConfigurationModal
		return m, nil
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenFuzzy):
		if len(m.ui.Config.Projects) > 0 {
			m.ui.ViewState = model.StateFuzzyModal
			m.ui.FuzzyMode = model.FuzzyModeFiles
			m.ui.FuzzyQueryInput.SetValue("")
			m.ui.FuzzyResults = []model.FuzzyResult{}
			m.ui.SelectedFuzzy = 0
			m.ui.FuzzyQueryInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenDocker):
		m.ui.ViewState = model.StateDockerModal
		m.ui.SelectedDockerRow = 0
		return m, m.fetchDockerContainersCmd()

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenJdk):
		m.ui.PreviousViewState = model.StateDashboard
		return m.prepareProfileScreen(model.StateJDKConfigModal, model.JdkName, model.JdkPath)

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenMvn):
		m.ui.PreviousViewState = model.StateDashboard
		return m.prepareProfileScreen(model.StateMavenConfigModal, model.MvnName, model.MvnPath)

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenConfiguration):
		m.ui.ViewState = model.StateConfigDeckModal
		m.ui.SelectedConfigOption = 0
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenBuild):
		if len(m.ui.Config.Projects) > 0 {
			m.ui.ViewState = model.StateBuildModal
			m.ui.SelectedBuildOption = 0
			m.ui.BuildLogs = []string{"Console engine ready. Select command step to initialize stream..."}
			m.ui.IsBuilding = false
			return m, nil
		}
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenSession):
		m.ui.ViewState = model.StateSessionLogsModal
		m.ui.ViewingSessionID = 0
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenHelp):
		m.ui.ViewState = model.StateHelpModal
		return m, nil

	case kbd.Tab:
		m.ui.ActiveFocus = model.FocusArea((int(m.ui.ActiveFocus) + 1) % 3)
		return m, nil

	case "up", "k":
		if m.ui.ActiveFocus == model.FocusProjects && m.ui.SelectedProject > 0 {
			m.ui.SelectedProject--
			m.refreshRightPaneFromSelectedProject()
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.ui.ActiveFocus == model.FocusTree && m.ui.SelectedFile > 0 {
			m.ui.SelectedFile--
			cmds = append(cmds, m.readFileContentCmd())
		}

	case "down", "j":
		if m.ui.ActiveFocus == model.FocusProjects && m.ui.SelectedProject < len(m.ui.Config.Projects)-1 {
			m.ui.SelectedProject++
			m.refreshRightPaneFromSelectedProject()
			cmds = append(cmds, m.updateWorkspaceFiles())
		} else if m.ui.ActiveFocus == model.FocusTree && m.ui.SelectedFile < len(m.ui.TreeNodes)-1 {
			m.ui.SelectedFile++
			cmds = append(cmds, m.readFileContentCmd())
		}

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save), "right", "l":
		if m.ui.ActiveFocus == model.FocusMenu {
			cmds = append(cmds, m.executeActiveMenuAction())
		} else if m.ui.ActiveFocus == model.FocusTree && len(m.ui.TreeNodes) > 0 {
			idx := m.ui.SelectedFile
			if idx >= 0 && idx < len(m.ui.TreeNodes) {
				if m.ui.TreeNodes[idx].IsDir {
					m.ui.TreeNodes[idx].IsExpanded = !m.ui.TreeNodes[idx].IsExpanded
					m.rebuildActiveTree()
				} else {
					cmds = append(cmds, m.readFileContentCmd())
				}
			}
		}

	case "left", "backspace":
		if m.ui.ActiveFocus == model.FocusTree && len(m.ui.TreeNodes) > 0 {
			idx := m.ui.SelectedFile
			if idx >= 0 && idx < len(m.ui.TreeNodes) {
				if m.ui.TreeNodes[idx].IsDir && m.ui.TreeNodes[idx].IsExpanded {
					m.ui.TreeNodes[idx].IsExpanded = false
					m.rebuildActiveTree()
				}
			}
		}

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.OpenEditFile):
		m.ui.ViewState = model.StateEditorModal
		m.ui.FocusedInput = model.EditContent
		return m, m.loadFileCmd()
	}

	return m, tea.Batch(cmds...)
}
