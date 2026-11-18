package componentaction

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

// SessionLogsModal is the session inspector: up/down selects, enter opens
// the session in the terminal pane (live), d closes it (killing it if it is
// still running).
//
//goland:noinspection GoMixedReceiverTypes
func SessionLogsModal(ui *model.UI, msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == action.GetShortcutKeyBinding(ui.Config.ShortCuts, action.Escape) || ui.Shells == nil {
		ui.ViewState = model.StateDashboard
		return nil
	}
	sessions := ui.Shells.List()
	if len(sessions) == 0 {
		return nil
	}
	ui.SelectedSession = min(ui.SelectedSession, len(sessions)-1)

	switch msg.String() {
	case "up", "k":
		ui.SelectedSession = max(ui.SelectedSession-1, 0)
	case "down", "j":
		ui.SelectedSession = min(ui.SelectedSession+1, len(sessions)-1)
	case "enter":
		AttachSession(ui, ui.Shells, sessions[ui.SelectedSession].ID)
	case "d":
		id := sessions[ui.SelectedSession].ID
		_ = ui.Shells.Close(id)
		if i := slices.Index(ui.TerminalTabs, id); i >= 0 {
			ui.TerminalTabs = slices.Delete(ui.TerminalTabs, i, i+1)
			ui.ActiveTerminalTab = max(min(ui.ActiveTerminalTab, len(ui.TerminalTabs)-1), 0)
		}
		ui.SelectedSession = max(min(ui.SelectedSession, len(sessions)-2), 0)
	}
	return nil
}
