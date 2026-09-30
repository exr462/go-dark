package componentaction

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func JDKModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch ui.JDKStep {
	case model.StepSelectJDKAction:
		switch msg.String() {
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
			ui.ViewState = ui.PreviousViewState
			return nil
		case "up", "k":
			if ui.SelectedMenuIndex > 0 {
				ui.SelectedMenuIndex--
			}
		case "down", "j":
			if ui.SelectedMenuIndex < 1 {
				ui.SelectedMenuIndex++
			}
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
			if ui.SelectedMenuIndex == 0 {
				ui.JDKStep = model.StepAddNewJDKVersion
				initFields(ui, kbd.JdkName, kbd.JdkPath)
			} else {
				ui.JDKStep = model.StepAssignJDKToProject
				ui.SelectedJDKIndex = 0
			}
		}
	case model.StepAddNewJDKVersion:
		switch msg.String() {
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
			ui.JDKStep = model.StepSelectJDKAction
			return nil
		case "tab", "down":
			switcheroo(ui, kbd.JdkName, kbd.JdkPath)
		case "shift+tab", "up":
			switcheroo(ui, kbd.JdkName, kbd.JdkPath)
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
			jName, jPath := getValues(ui, kbd.JdkName, kbd.JdkPath)
			if jName != "" && jPath != "" {
				ui.Config.JDKs = append(ui.Config.JDKs, config.Profile{Name: jName, Path: jPath})
				_ = config.SaveConfig(ui.Config)
				ui.StatusMsg = fmt.Sprintf("✅ Added Java Profile: %s", jName)
			}
			ui.JDKStep = model.StepSelectJDKAction
			return nil
		}
		var cmd tea.Cmd
		ui.Inputs[ui.FocusedInput], cmd = ui.Inputs[ui.FocusedInput].Update(msg)
		return cmd
	case model.StepAssignJDKToProject:
		switch msg.String() {
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
			ui.JDKStep = model.StepSelectJDKAction
			return nil
		case "up", "k":
			if ui.SelectedJDKIndex > 0 {
				ui.SelectedJDKIndex--
			}
		case "down", "j":
			if ui.SelectedJDKIndex < len(ui.Config.JDKs)-1 {
				ui.SelectedJDKIndex++
			}
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
			if len(ui.Config.JDKs) > 0 && len(ui.Config.Projects) > 0 {
				chosenJDK := ui.Config.JDKs[ui.SelectedJDKIndex]
				ui.Config.Projects[ui.SelectedProject].JDKName = chosenJDK.Name
				_ = config.SaveConfig(ui.Config)
				ui.StatusMsg = fmt.Sprintf("✅ Assigned JDK Environment: %s", chosenJDK.Name)
			}
			ui.ViewState = ui.PreviousViewState
			return initializer.InitializeWorkspace(ui).OnAction()
		}
	}
	return nil
}
