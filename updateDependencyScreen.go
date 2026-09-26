package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateDependencyScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	projIdx := m.state.DepScreen.ActiveProjectIndex
	if projIdx < 0 || projIdx >= len(m.state.Config.Projects) {
		m.state.ViewState = model.StateDashboard
		return m, nil
	}

	// 🔍 FIX 1: Direct slice reference modification
	// Do NOT copy out the project struct into a local variable.
	// Instead, manipulate the array directly at its absolute index position.

	switch msg.String() {
	case "up", "k":
		if m.state.DepScreen.Cursor > 0 {
			m.state.DepScreen.Cursor--
		}

	case "down", "j":
		if m.state.DepScreen.Cursor < len(m.state.DepScreen.AvailableOptions)-1 {
			m.state.DepScreen.Cursor++
		}

	case "space":
		selectedTarget := m.state.DepScreen.AvailableOptions[m.state.DepScreen.Cursor]

		// Find if dependency exists in our absolute index target
		foundIdx := -1
		for i, dep := range config.AvailableProjects[projIdx].Dependencies {
			if dep == selectedTarget {
				foundIdx = i
				break
			}
		}

		if foundIdx >= 0 {
			// Toggle Off: Remove dependency directly from the slice array
			config.AvailableProjects[projIdx].Dependencies = append(
				config.AvailableProjects[projIdx].Dependencies[:foundIdx],
				config.AvailableProjects[projIdx].Dependencies[foundIdx+1:]...,
			)
		} else {
			// Toggle On: Add dependency directly to the slice array
			config.AvailableProjects[projIdx].Dependencies = append(
				config.AvailableProjects[projIdx].Dependencies,
				selectedTarget,
			)
		}

	case "enter":
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()

	case "esc", "q":
		m.state.ViewState = model.StateDashboard
		return m, nil
	}

	// 🔍 FIX 2: Return the updated model 'm' back to Bubble Tea!
	// If you were returning 'nil, nil' or a raw unmutated model state here,
	// Bubble Tea wouldn't know the selection markers changed.
	return m, nil
}
