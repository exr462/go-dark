package componentaction

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

// OpenShortcutsConfiguration fills the shortcut inputs from the config and
// shows the shortcuts screen, returning to returnTo on exit.
func OpenShortcutsConfiguration(ui *model.UI, returnTo model.ApplicationViewState) {
	ui.PreviousViewState = returnTo
	kbd.LoadShortcutsOnInputs(ui.Config.ShortCuts, ui.Inputs)
	for _, f := range kbd.ShortcutFields {
		ui.Inputs[f.Field].Blur()
	}
	ui.FocusedInput = kbd.ShortcutFields[0].Field
	ui.Inputs[ui.FocusedInput].Focus()
	ui.ViewState = model.StateShortcutConfigurationModal
}

// shortcutFieldIndex is the position of the focused input in kbd.ShortcutFields.
func shortcutFieldIndex(field kbd.InputField) int {
	for i, f := range kbd.ShortcutFields {
		if f.Field == field {
			return i
		}
	}
	return 0
}

//goland:noinspection GoMixedReceiverTypes
func ShortcutsConfigurationModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		// returning to modal
		ui.ViewState = ui.PreviousViewState
		return nil

	case "tab", "down":
		i := min(shortcutFieldIndex(ui.FocusedInput)+1, len(kbd.ShortcutFields)-1)
		focusShortcutField(ui, i)
		return nil

	case "shift+tab", "up":
		i := max(shortcutFieldIndex(ui.FocusedInput)-1, 0)
		focusShortcutField(ui, i)
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

func focusShortcutField(ui *model.UI, i int) {
	ui.Inputs[ui.FocusedInput].Blur()
	ui.FocusedInput = kbd.ShortcutFields[i].Field
	ui.Inputs[ui.FocusedInput].Focus()
}
