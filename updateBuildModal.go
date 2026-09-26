package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) updateBuildModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.state.Config.Projects) == 0 {
		m.state.ViewState = model.StateDashboard
		return m, nil
	}

	targetProj := m.state.Config.Projects[m.state.SelectedProject]
	var currentSessionID int
	for id, sess := range m.state.Sessions {
		if sess.ProjectName == targetProj.Name && sess.IsRunning {
			currentSessionID = id
			break
		}
	}

	if currentSessionID != 0 {
		if msg.String() == "esc" {
			m.state.ViewState = model.StateDashboard
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "left", "h":
		if m.state.SelectedBuildOption > 0 {
			m.state.SelectedBuildOption--
		}
	case "right", "l":
		if m.state.SelectedBuildOption < len(m.state.BuildOptions)-1 {
			m.state.SelectedBuildOption++
		}
	case "enter":
		m.state.IsBuilding = true
		chosenOpt := m.state.BuildOptions[m.state.SelectedBuildOption]
		return m, m.spawnBackgroundSession(targetProj, chosenOpt)
	}
	return m, nil
}
