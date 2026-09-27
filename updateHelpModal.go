package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateHelpModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape),
		action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.OpenHelp),
		action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.QuitApplication),
		action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
		m.state.ViewState = model.StateDashboard
		return m, nil
	}
	return m, nil
}
