package main

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateGitConfiguration(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "tab", "down":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput = (m.state.FocusedInput + 1) % 4
		m.state.Inputs[m.state.FocusedInput].Focus()
		return m, nil
	case "shift+tab", "up":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput--
		if m.state.FocusedInput < 0 {
			m.state.FocusedInput = 2
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
		return m, nil
	case "enter":
		m.state.Config.BasePath = m.state.Inputs[model.GitWorkspace].Value()
		m.state.Config.GitUsername = m.state.Inputs[model.GitUsername].Value()
		m.state.Config.GitEmail = m.state.Inputs[model.GitEmail].Value()
		m.state.Config.MaxListTag, _ = strconv.Atoi(m.state.Inputs[model.GitMaxTagListSize].Value())
		if m.state.Config.BasePath == "" {
			return m, nil
		}
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, nil
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
