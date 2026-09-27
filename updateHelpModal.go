package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateHelpModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case config.GetKeyBinding(m.state.Config.ShortCuts, config.CancelKeyBind),
		config.GetKeyBinding(m.state.Config.ShortCuts, config.HelpKeyBind),
		config.GetKeyBinding(m.state.Config.ShortCuts, config.QuitKeyBind),
		config.GetKeyBinding(m.state.Config.ShortCuts, config.SubmitKeyBind):
		m.state.ViewState = model.StateDashboard
		return m, nil
	}
	return m, nil
}
