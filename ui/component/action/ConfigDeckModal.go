package componentaction

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func ConfigDeckModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.ViewState = model.StateDashboard
		return nil

	case "up", "k":
		if ui.SelectedConfigOption > 0 {
			ui.SelectedConfigOption--
		}
		return nil

	case "down", "j":
		if ui.SelectedConfigOption < 3 {
			ui.SelectedConfigOption++
		}
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		ui.PreviousViewState = model.StateConfigDeckModal
		switch ui.SelectedConfigOption {
		case 0:
			ui.ViewState = model.StateGitConfigurationModal
			ui.FocusedInput = kbd.GitWorkspace
			ui.Inputs[kbd.GitWorkspace].SetValue(ui.Config.BasePath)
			ui.Inputs[kbd.GitUsername].SetValue(ui.Config.GitUsername)
			ui.Inputs[kbd.GitEmail].SetValue(ui.Config.GitEmail)
			ui.Inputs[kbd.GitMaxTagListSize].SetValue(strconv.Itoa(ui.Config.MaxListTag))
			ui.Inputs[kbd.GitWorkspace].Focus()
			return textinput.Blink

		case 1:
			ui.ViewState = model.StateJDKConfigModal
			ui.FocusedInput = kbd.JdkName
			ui.JDKStep = model.StepSelectJDKAction
			ui.Inputs[kbd.JdkName].SetValue("")
			ui.Inputs[kbd.JdkPath].SetValue("")
			ui.SelectedMenuIndex = 0
			return nil

		case 2:
			ui.ViewState = model.StateMavenConfigModal
			ui.FocusedInput = kbd.MvnName
			ui.MavenStep = model.StepSelectMvnAction
			ui.Inputs[kbd.MvnName].SetValue("")
			ui.Inputs[kbd.MvnPath].SetValue("")
			ui.SelectedMenuIndex = 0
			return nil

		case 3:
			ui.ViewState = model.StateShortcutConfigurationModal
			ui.FocusedInput = kbd.FuzzyKey
			kbd.LoadShortcutsOnConfig(ui.Config.ShortCuts, ui.Inputs)
			ui.Inputs[kbd.FuzzyKey].Focus()
			// Save dynamic values safely across memory pointers
			ui.SelectedMenuIndex = 0
			return nil
		}
	}
	return nil
}
