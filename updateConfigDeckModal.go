package main

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateConfigDeckModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
		m.ui.ViewState = model.StateDashboard
		return m, nil

	case "up", "k":
		if m.ui.SelectedConfigOption > 0 {
			m.ui.SelectedConfigOption--
		}
		return m, nil

	case "down", "j":
		if m.ui.SelectedConfigOption < 3 {
			m.ui.SelectedConfigOption++
		}
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
		m.ui.PreviousViewState = model.StateConfigDeckModal
		switch m.ui.SelectedConfigOption {
		case 0:
			m.ui.ViewState = model.StateGitConfigurationModal
			m.ui.FocusedInput = model.GitWorkspace
			m.ui.Inputs[model.GitWorkspace].SetValue(m.ui.Config.BasePath)
			m.ui.Inputs[model.GitUsername].SetValue(m.ui.Config.GitUsername)
			m.ui.Inputs[model.GitEmail].SetValue(m.ui.Config.GitEmail)
			m.ui.Inputs[model.GitMaxTagListSize].SetValue(strconv.Itoa(m.ui.Config.MaxListTag))
			m.ui.Inputs[model.GitWorkspace].Focus()
			return m, textinput.Blink

		case 1:
			m.ui.ViewState = model.StateJDKConfigModal
			m.ui.FocusedInput = model.JdkName
			m.ui.JDKStep = model.StepSelectJDKAction
			m.ui.Inputs[model.JdkName].SetValue("")
			m.ui.Inputs[model.JdkPath].SetValue("")
			m.ui.SelectedMenuIndex = 0
			return m, nil

		case 2:
			m.ui.ViewState = model.StateMavenConfigModal
			m.ui.FocusedInput = model.MvnName
			m.ui.MavenStep = model.StepSelectMvnAction
			m.ui.Inputs[model.MvnName].SetValue("")
			m.ui.Inputs[model.MvnPath].SetValue("")
			m.ui.SelectedMenuIndex = 0
			return m, nil

		case 3:
			m.ui.ViewState = model.StateShortcutConfigurationModal
			m.ui.FocusedInput = model.FuzzyKey
			m.loadShortcutsOnInputs()
			m.ui.Inputs[model.FuzzyKey].Focus()
			// Save dynamic values safely across memory pointers
			m.ui.SelectedMenuIndex = 0
			return m, nil
		}
	}
	return m, nil
}
