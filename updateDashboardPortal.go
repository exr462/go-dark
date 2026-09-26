package main

import (
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateDashboardPortal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "ctrl+g":
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
	case "ctrl+f":
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

	case "ctrl+d":
		m.state.ViewState = model.StateDockerModal
		m.state.SelectedDockerRow = 0
		return m, m.fetchDockerContainersCmd()

	case "ctrl+j":
		m.state.ViewState = model.StateJDKConfigModal
		m.state.JDKStep = model.StepSelectJDKAction
		m.state.SelectedMenuIndex = 0
		m.state.SelectedJDKIndex = 0
		return m, nil

	case "ctrl+u":
		m.state.ViewState = model.StateMavenConfigModal
		m.state.MavenStep = model.StepSelectMvnAction
		m.state.SelectedMenuIndex = 0
		m.state.SelectedMavenIndex = 0
		return m, nil

	case "ctrl+y":
		m.state.ViewState = model.StateConfigDeckModal
		m.state.SelectedConfigOption = 0
		return m, nil

	case "ctrl+n":
		m.state.ViewState = model.StateAddProjectModal
		m.state.FocusedInput = model.ProjectName
		m.state.Inputs[model.ProjectName].SetValue("")
		m.state.Inputs[model.RelativeFolder].SetValue("")
		m.state.Inputs[model.StackType].SetValue("")
		m.state.Inputs[model.GitCloneURL].SetValue("")
		m.state.Inputs[model.ProjectName].Focus()
		return m, textinput.Blink

	case "ctrl+b":
		if len(m.state.Config.Projects) > 0 {
			m.state.ViewState = model.StateBuildModal
			m.state.SelectedBuildOption = 0
			m.state.BuildLogs = []string{"Console engine ready. Select command step to initialize stream..."}
			m.state.IsBuilding = false
			return m, nil
		}
		return m, nil

	case "ctrl+s":
		m.state.ViewState = model.StateSessionLogsModal
		m.state.ViewingSessionID = 0
		return m, nil

	case "?", "h":
		m.state.ViewState = model.StateHelpModal
		return m, nil

	case "tab":
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

	case "enter", "right", "l":
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

	case "ctrl+e":
		if len(m.state.TreeNodes) == 0 || m.state.SelectedFile < 0 || m.state.SelectedFile >= len(m.state.TreeNodes) {
			return m, nil
		}
		selectedNode := m.state.TreeNodes[m.state.SelectedFile]
		if selectedNode.IsDir {
			return m, nil
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nvim"
		}
		c := exec.Command(editor, selectedNode.FullPath)
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return model.FileLoadMsg("sync")
		})
	}

	return m, tea.Batch(cmds...)
}
