package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateDockerModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "up", "k":
		if m.state.SelectedDockerRow > 0 {
			m.state.SelectedDockerRow--
		}
	case "down", "j":
		if m.state.SelectedDockerRow < len(m.state.DockerContainers)-1 {
			m.state.SelectedDockerRow++
		}
	case "s", "t", "r":
		if len(m.state.DockerContainers) == 0 || m.state.SelectedDockerRow >= len(m.state.DockerContainers) {
			return m, nil
		}
		target := m.state.DockerContainers[m.state.SelectedDockerRow]
		action := "start"
		if msg.String() == "t" {
			action = "stop"
		}
		if msg.String() == "r" {
			action = "restart"
		}
		return m, tea.Batch(m.runDockerActionCmd(target.ID, action), m.fetchDockerContainersCmd())
	}
	return m, nil
}
