package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func ShortcutsConfigurationModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		// returning to modal
		ui.ViewState = ui.PreviousViewState
		return nil

	case "tab", "down":
		ui.Inputs[ui.FocusedInput].Blur()
		// prevents us to trigger the blinker out of screen
		if ui.FocusedInput > kbd.ToggleKey {
			ui.FocusedInput = kbd.ToggleKey
		}
		ui.FocusedInput = ui.FocusedInput + 1
		ui.Inputs[ui.FocusedInput].Focus()
		return nil

	case "shift+tab", "up":
		ui.Inputs[ui.FocusedInput].Blur()
		ui.FocusedInput--
		// prevents us to trigger the blinker out of screen
		if ui.FocusedInput < kbd.FuzzyKey {
			ui.FocusedInput = kbd.FuzzyKey
		}

		ui.Inputs[ui.FocusedInput].Focus()
		return nil

	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		kbd.LoadShortcutsOnConfig(ui.Config.ShortCuts, ui.Inputs)
		_ = config.SaveConfig(ui.Config)
		ui.ViewState = ui.PreviousViewState
		return nil
	}
	var cmd tea.Cmd
	ui.Inputs[ui.FocusedInput], cmd = ui.Inputs[ui.FocusedInput].Update(msg)
	return cmd
}
