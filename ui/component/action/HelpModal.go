package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func HelpModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape),
		action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.OpenHelp),
		action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.QuitApplication),
		action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Save):
		ui.ViewState = model.StateDashboard
		return nil
	}
	return nil
}
