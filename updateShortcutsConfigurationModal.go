package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateShortcutsConfigurationModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
		// returning to modal
		m.ui.ViewState = m.ui.PreviousViewState
		return m, nil

	case "tab", "down":
		m.ui.Inputs[m.ui.FocusedInput].Blur()
		// prevents us to trigger the blinker out of screen
		if m.ui.FocusedInput > model.ToggleKey {
			m.ui.FocusedInput = model.ToggleKey
		}
		m.ui.FocusedInput = m.ui.FocusedInput + 1
		m.ui.Inputs[m.ui.FocusedInput].Focus()
		return m, nil

	case "shift+tab", "up":
		m.ui.Inputs[m.ui.FocusedInput].Blur()
		m.ui.FocusedInput--
		// prevents us to trigger the blinker out of screen
		if m.ui.FocusedInput < model.FuzzyKey {
			m.ui.FocusedInput = model.FuzzyKey
		}

		m.ui.Inputs[m.ui.FocusedInput].Focus()
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
		m.loadShortcutsOnConfig()
		_ = config.SaveConfig(m.ui.Config)
		m.ui.ViewState = m.ui.PreviousViewState
		return m, nil
	}
	var cmd tea.Cmd
	m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
	return m, cmd
}
