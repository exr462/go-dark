package componentaction

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func SessionLogsModal(ui *model.UI, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape):
		ui.ViewState = model.StateDashboard
		return nil
	default:
		if msg.String() >= "1" && msg.String() <= "9" {
			runes := []rune(msg.String())
			if len(runes) > 0 {
				targetID := int(runes[0] - '0')
				ui.ViewingSessionID = targetID
				return nil
			}
		}
	}
	return nil
}
