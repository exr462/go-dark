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
func MvnModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	//goland:noinspection DuplicatedCode
	switch ui.MavenStep {
	case model.StepSelectMvnAction:
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
				ui.MavenStep = model.StepAddNewMvnVersion
				initFields(ui, kbd.MvnName, kbd.MvnPath)
			} else {
				ui.MavenStep = model.StepAssignMvnToProject
				ui.SelectedMavenIndex = 0
			}
		}
	case model.StepAddNewMvnVersion:
		switch msg.String() {
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
			ui.MavenStep = model.StepSelectMvnAction
			return nil
		case "tab", "down":
			switcheroo(ui, kbd.MvnName, kbd.MvnPath)
		case "shift+tab", "up":
			switcheroo(ui, kbd.MvnName, kbd.MvnPath)
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
			mvnName, mvnPath := getValues(ui, kbd.MvnName, kbd.MvnPath)
			if mvnName != "" && mvnPath != "" {
				ui.Config.Mavens = append(ui.Config.Mavens, config.Profile{Name: mvnName, Path: mvnPath})
				_ = config.SaveConfig(ui.Config)
				ui.StatusMsg = fmt.Sprintf("✅ Added Maven Profile: %s", mvnName)
			}
			ui.MavenStep = model.StepSelectMvnAction
			return nil
		}
		var cmd tea.Cmd
		ui.Inputs[ui.FocusedInput], cmd = ui.Inputs[ui.FocusedInput].Update(msg)
		return cmd
	case model.StepAssignMvnToProject:
		switch msg.String() {
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
			ui.MavenStep = model.StepSelectMvnAction
			return nil
		case "up", "k":
			if ui.SelectedMavenIndex > 0 {
				ui.SelectedMavenIndex--
			}
		case "down", "j":
			if ui.SelectedMavenIndex < len(ui.Config.Mavens)-1 {
				ui.SelectedMavenIndex++
			}
		case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
			if len(ui.Config.Mavens) > 0 && len(ui.Config.Projects) > 0 {
				chosenMvn := ui.Config.Mavens[ui.SelectedMavenIndex]
				ui.Config.Projects[ui.SelectedProject].MavenName = chosenMvn.Name
				_ = config.SaveConfig(ui.Config)
				ui.StatusMsg = fmt.Sprintf("✅ Assigned Maven Profile: %s", chosenMvn.Name)
			}
			ui.ViewState = ui.PreviousViewState
			return initializer.InitializeWorkspace(ui).OnAction()
		}
	}
	return nil
}

func initFields(ui *model.UI, name kbd.InputField, path kbd.InputField) {
	ui.FocusedInput = name
	ui.Inputs[name].SetValue("")
	ui.Inputs[path].SetValue("")
	ui.Inputs[name].Focus()
}

func switcheroo(ui *model.UI, name kbd.InputField, path kbd.InputField) {
	ui.Inputs[ui.FocusedInput].Blur()
	if ui.FocusedInput == name {
		ui.FocusedInput = path
	} else if ui.FocusedInput == path {
		ui.FocusedInput = name
	}
	ui.Inputs[ui.FocusedInput].Focus()
}

func getValues(ui *model.UI, name kbd.InputField, path kbd.InputField) (string, string) {
	mVnName := ui.Inputs[name].Value()
	mVnPath := ui.Inputs[path].Value()
	return mVnName, mVnPath
}
