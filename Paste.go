package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/exr462/go-dark/fuzzy"
	"github.com/exr462/go-dark/kbd"
	"github.com/exr462/go-dark/model"
)

// routePaste forwards a bracketed paste to whichever text component the
// active screen is editing. Bubble Tea v2 delivers pastes as tea.PasteMsg
// instead of a tea.KeyMsg, so the key-based modal handlers never see them.
func (m *appModel) routePaste(msg tea.PasteMsg) tea.Cmd {
	var cmd tea.Cmd
	switch m.ui.ViewState {
	case model.StateEditorModal:
		if m.ui.EditorMode != model.EditorModeInsert {
			return nil
		}
		m.ui.Editor, cmd = m.ui.Editor.Update(msg)
		m.ui.EditorDirty = m.ui.Editor.Value() != m.ui.EditorOriginalContent
	case model.StateFuzzyModal:
		oldVal := m.ui.FuzzyQueryInput.Value()
		m.ui.FuzzyQueryInput, cmd = m.ui.FuzzyQueryInput.Update(msg)
		if m.ui.FuzzyQueryInput.Value() != oldVal {
			m.ui.SelectedFuzzy = 0
			fuzzy.FuzzySearchEngine(m.ui)
			fuzzy.FuzzyPreviewPane(m.ui)
		}
	case model.StateTerminalCockpit:
		m.ui.Inputs[kbd.Terminal], cmd = m.ui.Inputs[kbd.Terminal].Update(msg)
	case model.StateGitConfigurationModal, model.StateShortcutConfigurationModal:
		m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
	case model.StateTeamsModal:
		if m.ui.AddingTeam {
			m.ui.Inputs[kbd.TeamGitURL], cmd = m.ui.Inputs[kbd.TeamGitURL].Update(msg)
		}
	case model.StateProjectsModal:
		if m.ui.AddingProject && m.ui.FocusedInput >= kbd.ProjectName && m.ui.FocusedInput <= kbd.ProjectPath {
			m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
		}
	case model.StateJDKConfigModal:
		if m.ui.JDKStep == model.StepAddNewJDKVersion {
			m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
		}
	case model.StateMavenConfigModal:
		if m.ui.MavenStep == model.StepAddNewMvnVersion {
			m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
		}
	}
	return cmd
}
