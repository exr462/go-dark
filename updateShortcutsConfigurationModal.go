package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateShortcutsConfigurationModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case config.GetShortcutKeyBinding(m.state.Config.ShortCuts, config.CancelKeyBind):
		// returning to modal
		m.state.ViewState = m.state.PreviousViewState
		return m, nil

	case "tab", "down":
		m.state.Inputs[m.state.FocusedInput].Blur()
		// prevents us to trigger the blinker out of screen
		if m.state.FocusedInput > model.ToggleKey {
			m.state.FocusedInput = model.ToggleKey
		}
		m.state.FocusedInput = m.state.FocusedInput + 1
		m.state.Inputs[m.state.FocusedInput].Focus()
		return m, nil

	case "shift+tab", "up":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput--
		// prevents us to trigger the blinker out of screen
		if m.state.FocusedInput < model.FuzzyKey {
			m.state.FocusedInput = model.FuzzyKey
		}

		m.state.Inputs[m.state.FocusedInput].Focus()
		return m, nil

	case config.GetShortcutKeyBinding(m.state.Config.ShortCuts, config.SubmitKeyBind):
		m.loadShortcutsOnConfig()
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = m.state.PreviousViewState
		return m, nil
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
