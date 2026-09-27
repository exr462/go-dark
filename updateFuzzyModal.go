package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateFuzzyModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
		m.ui.ViewState = model.StateDashboard
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Toggle):
		if m.ui.FuzzyMode == model.FuzzyModeFiles {
			m.ui.FuzzyMode = model.FuzzyModeContent
		} else {
			m.ui.FuzzyMode = model.FuzzyModeFiles
		}
		m.ui.SelectedFuzzy = 0
		m.runFuzzySearchEngine()
		m.syncFuzzyPreviewPane()
		return m, nil

	case "up", "k":
		if m.ui.SelectedFuzzy > 0 {
			m.ui.SelectedFuzzy--
			m.syncFuzzyPreviewPane()
		}
		return m, nil

	case "down", "j":
		if m.ui.SelectedFuzzy < len(m.ui.FuzzyResults)-1 {
			m.ui.SelectedFuzzy++
			m.syncFuzzyPreviewPane()
		}
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
		if len(m.ui.FuzzyResults) > 0 && m.ui.SelectedFuzzy < len(m.ui.FuzzyResults) {
			target := m.ui.FuzzyResults[m.ui.SelectedFuzzy]
			m.ui.ViewState = model.StateDashboard

			for idx, node := range m.ui.TreeNodes {
				if node.FullPath == target.FullPath {
					m.ui.SelectedFile = idx
					m.ui.ActiveFocus = model.FocusTree
					break
				}
			}
			return m, m.readFileContentCmd()
		}
		m.ui.ViewState = model.StateDashboard
		return m, nil
	}

	var cmd tea.Cmd
	oldVal := m.ui.FuzzyQueryInput.Value()
	var genericMsg tea.Msg = msg
	m.ui.FuzzyQueryInput, cmd = m.ui.FuzzyQueryInput.Update(genericMsg)

	if m.ui.FuzzyQueryInput.Value() != oldVal {
		m.ui.SelectedFuzzy = 0
		m.runFuzzySearchEngine()
		m.syncFuzzyPreviewPane()
	}

	return m, cmd
}
