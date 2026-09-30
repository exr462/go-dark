package componentaction

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	commandfile "github.com/exr462/go-dark/command/file"
	commandpanel "github.com/exr462/go-dark/command/panel"
	"github.com/exr462/go-dark/docker"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/lsp"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/state"
	"github.com/exr462/go-dark/terminal"
)

func profileScreen(ui *model.UI, viewState model.ApplicationViewState, name kbd.InputField, path kbd.InputField) tea.Cmd {
	if viewState == model.StateJDKConfigModal {
		ui.JDKStep = model.StepSelectJDKAction
	} else {
		ui.MavenStep = model.StepSelectMvnAction
	}
	ui.ViewState = viewState
	ui.Inputs[name].SetValue("")
	ui.Inputs[path].SetValue("")
	ui.SelectedMenuIndex = 0
	ui.SelectedJDKIndex = 0
	return nil
}

//goland:noinspection GoMixedReceiverTypes
func DashboardPortal(ui *model.UI, providerFactory *lsp.ProviderFactory, msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.QuitApplication):
		return tea.Quit

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenGitOperations):
		if len(ui.Config.Projects) > 0 {
			ui.ViewState = model.StateGitOperationsModal
			ui.GitOperationStep = model.StepSelectGitProject
			ui.SelectedGitProject = ui.SelectedProject
			ui.SelectedGitCommand = 0
			return nil
		}
		return nil
	case "d": // 👈 Pressing 'd' over a highlighted row opens its dependency mapper
		if len(ui.Config.Projects) > 0 && ui.SelectedProject >= 0 {
			activeProj := ui.Config.Projects[ui.SelectedProject]

			// Initialize options excluding self to prevent cyclical dependencies
			ui.DepScreen = model.NewDependencyScreen(activeProj.Name, ui.Config.Projects)
			ui.DepScreen.ActiveProjectIndex = ui.SelectedProject

			// Flip view state boundary to render configuration modal
			ui.ViewState = model.StateDependencyConfigModal
			return nil
		}
		// Add this case inside your updateDashboardPortal switch-case handler:
	case "P": // 👈 Pressing Shift+P triggers the parallel DAG build
		ui.StatusMsg = "🏗️ Resolving dependency graph and starting parallel builds..."

		// Fire off the background compilation using your exact AvailableProjects layout
		// Limits the machine execution block to a safe ceiling of 4 concurrent threads
		return initializer.InitializeTriggerPipeline(ui).OnAction()

	case "B": // 👈 Capital 'B' triggers the full parallel build pipeline
		ui.StatusMsg = "🏗️ Initializing parallel build graph..."
		ui.ViewState = model.StateBuildModal // Optional: switch to a loading/progress view

		// Pass your max concurrency limit (e.g., 4 simultaneous builds)
		return initializer.InitializeTriggerPipeline(ui).OnAction()
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenEditShortcuts):
		ui.PreviousViewState = model.StateDashboard
		kbd.LoadShortcutsOnInputs(ui.Config.ShortCuts, ui.Inputs)
		ui.ViewState = model.StateShortcutConfigurationModal
		return nil
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenFuzzy):
		if len(ui.Config.Projects) > 0 {
			ui.ViewState = model.StateFuzzyModal
			ui.FuzzyMode = model.FuzzyModeFiles
			ui.FuzzyQueryInput.SetValue("")
			ui.FuzzyResults = []model.FuzzyResult{}
			ui.SelectedFuzzy = 0
			ui.FuzzyQueryInput.Focus()
			return textinput.Blink
		}
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenDocker):
		ui.ViewState = model.StateDockerModal
		ui.SelectedDockerRow = 0
		return docker.FetchDockerContainers()

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenJdk):
		ui.PreviousViewState = model.StateDashboard
		return profileScreen(ui, model.StateJDKConfigModal, kbd.JdkName, kbd.JdkPath)

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenMvn):
		ui.PreviousViewState = model.StateDashboard
		return profileScreen(ui, model.StateMavenConfigModal, kbd.MvnName, kbd.MvnPath)

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenConfiguration):
		ui.ViewState = model.StateConfigDeckModal
		ui.SelectedConfigOption = 0
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenBuild):
		if len(ui.Config.Projects) > 0 {
			ui.ViewState = model.StateBuildModal
			ui.SelectedBuildOption = 0
			ui.BuildLogs = []string{"Console engine ready. Select command step to initialize stream..."}
			ui.IsBuilding = false
			return nil
		}
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenSession):
		ui.ViewState = model.StateSessionLogsModal
		ui.ViewingSessionID = 0
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenHelp):
		ui.ViewState = model.StateHelpModal
		return nil

	case kbd.Tab:
		ui.ActiveFocus = model.FocusArea((int(ui.ActiveFocus) + 1) % 3)
		return nil

	case "up", "k":
		if ui.ActiveFocus == model.FocusProjects && ui.SelectedProject > 0 {
			ui.SelectedProject--
			commandpanel.RefreshSelectedProject(ui)
			cmds = append(cmds, initializer.InitializeWorkspace(ui).OnAction())
		} else if ui.ActiveFocus == model.FocusTree && ui.SelectedFile > 0 {
			ui.SelectedFile--
			cmds = append(cmds, commandfile.ReadFileContent(ui))
		}

	case "down", "j":
		if ui.ActiveFocus == model.FocusProjects && ui.SelectedProject < len(ui.Config.Projects)-1 {
			ui.SelectedProject++
			commandpanel.RefreshSelectedProject(ui)
			cmds = append(cmds, initializer.InitializeWorkspace(ui).OnAction())
		} else if ui.ActiveFocus == model.FocusTree && ui.SelectedFile < len(ui.TreeNodes)-1 {
			ui.SelectedFile++
			cmds = append(cmds, commandfile.ReadFileContent(ui))
		}

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save), "right", "l":
		if ui.ActiveFocus == model.FocusMenu {
			if len(ui.Config.Projects) == 0 {
				return nil
			}
			cmds = append(cmds, func() tea.Msg {
				return state.StatusMsg("Manual workspace isolation completed.")
			})
		} else if ui.ActiveFocus == model.FocusTree && len(ui.TreeNodes) > 0 {
			idx := ui.SelectedFile
			if idx >= 0 && idx < len(ui.TreeNodes) {
				if ui.TreeNodes[idx].IsDir {
					ui.TreeNodes[idx].IsExpanded = !ui.TreeNodes[idx].IsExpanded
					commandpanel.RebuildActiveTree(ui)
				} else {
					cmds = append(cmds, commandfile.ReadFileContent(ui))
				}
			}
		}

	case "left", "backspace":
		if ui.ActiveFocus == model.FocusTree && len(ui.TreeNodes) > 0 {
			idx := ui.SelectedFile
			if idx >= 0 && idx < len(ui.TreeNodes) {
				if ui.TreeNodes[idx].IsDir && ui.TreeNodes[idx].IsExpanded {
					ui.TreeNodes[idx].IsExpanded = false
					commandpanel.RebuildActiveTree(ui)
				}
			}
		}

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenEditFile):
		ui.ViewState = model.StateEditorModal
		ui.FocusedInput = kbd.EditContent
		return commandfile.LoadFileCmd(ui, providerFactory)
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenTerminal):
		ui.ViewState = model.StateTerminalCockpit
		ui.TerminalLogs = make([]terminal.LogLine, 0)
		ui.FocusedInput = kbd.Terminal
		return TerminalExecutionModal(ui, msg)
	}

	return tea.Batch(cmds...)
}
