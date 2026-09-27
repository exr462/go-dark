package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) updateBuildModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.ui.Config.Projects) == 0 {
		m.ui.ViewState = model.StateDashboard
		return m, nil
	}

	targetProj := m.ui.Config.Projects[m.ui.SelectedProject]
	var currentSessionID int
	for id, sess := range m.ui.Sessions {
		if sess.ProjectName == targetProj.Name && sess.IsRunning {
			currentSessionID = id
			break
		}
	}

	if currentSessionID != 0 {
		if msg.String() == "esc" {
			m.ui.ViewState = model.StateDashboard
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.ui.ViewState = model.StateDashboard
		return m, nil
	case "left", "h":
		if m.ui.SelectedBuildOption > 0 {
			m.ui.SelectedBuildOption--
		}
	case "right", "l":
		if m.ui.SelectedBuildOption < len(m.ui.BuildOptions)-1 {
			m.ui.SelectedBuildOption++
		}
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
		m.ui.IsBuilding = true
		chosenOpt := m.ui.BuildOptions[m.ui.SelectedBuildOption]
		return m, m.spawnBackgroundSession(targetProj, chosenOpt)
	}
	return m, nil
}
