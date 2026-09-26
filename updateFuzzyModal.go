package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateFuzzyModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil

	case "ctrl+t":
		if m.state.FuzzyMode == model.FuzzyModeFiles {
			m.state.FuzzyMode = model.FuzzyModeContent
		} else {
			m.state.FuzzyMode = model.FuzzyModeFiles
		}
		m.state.SelectedFuzzy = 0
		m.runFuzzySearchEngine()
		m.syncFuzzyPreviewPane()
		return m, nil

	case "up", "k":
		if m.state.SelectedFuzzy > 0 {
			m.state.SelectedFuzzy--
			m.syncFuzzyPreviewPane()
		}
		return m, nil

	case "down", "j":
		if m.state.SelectedFuzzy < len(m.state.FuzzyResults)-1 {
			m.state.SelectedFuzzy++
			m.syncFuzzyPreviewPane()
		}
		return m, nil

	case "enter":
		if len(m.state.FuzzyResults) > 0 && m.state.SelectedFuzzy < len(m.state.FuzzyResults) {
			target := m.state.FuzzyResults[m.state.SelectedFuzzy]
			m.state.ViewState = model.StateDashboard

			for idx, node := range m.state.TreeNodes {
				if node.FullPath == target.FullPath {
					m.state.SelectedFile = idx
					m.state.ActiveFocus = model.FocusTree
					break
				}
			}
			return m, m.readFileContentCmd()
		}
		m.state.ViewState = model.StateDashboard
		return m, nil
	}

	var cmd tea.Cmd
	oldVal := m.state.FuzzyQueryInput.Value()
	var genericMsg tea.Msg = msg
	m.state.FuzzyQueryInput, cmd = m.state.FuzzyQueryInput.Update(genericMsg)

	if m.state.FuzzyQueryInput.Value() != oldVal {
		m.state.SelectedFuzzy = 0
		m.runFuzzySearchEngine()
		m.syncFuzzyPreviewPane()
	}

	return m, cmd
}
