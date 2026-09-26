package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateModalForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.state.SelectedConfigOption == 3 {
			m.state.ViewState = model.StateConfigDeckModal
		} else {
			m.state.ViewState = model.StateDashboard
		}
		return m, nil
	case "tab", "down":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput++
		if m.state.FocusedInput > 7 {
			m.state.FocusedInput = 4
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "shift+tab", "up":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput--
		if m.state.FocusedInput < 4 {
			m.state.FocusedInput = 7
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "enter":
		name := m.state.Inputs[4].Value()
		path := m.state.Inputs[5].Value()
		pType := m.state.Inputs[6].Value()
		gitURL := m.state.Inputs[7].Value()
		if name == "" || path == "" {
			return m, nil
		}
		newProj := config.Project{
			Name:   name,
			Path:   config.ResolvePath(m.state.Config.BasePath, path),
			Type:   strings.ToLower(pType),
			GitURL: gitURL,
		}
		m.state.Config.Projects = append(m.state.Config.Projects, newProj)
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, func() tea.Msg { return model.ConfigRefreshedMsg(m.state.Config) }
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
