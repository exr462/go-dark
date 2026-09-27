package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateDockerModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
		m.ui.ViewState = model.StateDashboard
		return m, nil
	case "up", "k":
		if m.ui.SelectedDockerRow > 0 {
			m.ui.SelectedDockerRow--
		}
	case "down", "j":
		if m.ui.SelectedDockerRow < len(m.ui.DockerContainers)-1 {
			m.ui.SelectedDockerRow++
		}
	case "s", "t", "r":
		if len(m.ui.DockerContainers) == 0 || m.ui.SelectedDockerRow >= len(m.ui.DockerContainers) {
			return m, nil
		}
		target := m.ui.DockerContainers[m.ui.SelectedDockerRow]
		a := "start"
		if msg.String() == "t" {
			a = "stop"
		}
		if msg.String() == "r" {
			a = "restart"
		}
		return m, tea.Batch(m.runDockerActionCmd(target.ID, a), m.fetchDockerContainersCmd())
	}
	return m, nil
}
