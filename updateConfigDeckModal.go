package main

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateConfigDeckModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case config.GetKeyBinding(m.state.Config.ShortCuts, config.CancelKeyBind):
		m.state.ViewState = model.StateDashboard
		return m, nil

	case "up", "k":
		if m.state.SelectedConfigOption > 0 {
			m.state.SelectedConfigOption--
		}
		return m, nil

	case "down", "j":
		if m.state.SelectedConfigOption < 3 {
			m.state.SelectedConfigOption++
		}
		return m, nil

	case config.GetKeyBinding(m.state.Config.ShortCuts, config.SubmitKeyBind):
		m.state.PreviousViewState = model.StateConfigDeckModal
		switch m.state.SelectedConfigOption {
		case 0:
			m.state.ViewState = model.StateGitConfigurationModal
			m.state.FocusedInput = model.GitWorkspace
			m.state.Inputs[model.GitWorkspace].SetValue(m.state.Config.BasePath)
			m.state.Inputs[model.GitUsername].SetValue(m.state.Config.GitUsername)
			m.state.Inputs[model.GitEmail].SetValue(m.state.Config.GitEmail)
			m.state.Inputs[model.GitMaxTagListSize].SetValue(strconv.Itoa(m.state.Config.MaxListTag))
			m.state.Inputs[model.GitWorkspace].Focus()
			return m, textinput.Blink

		case 1:
			m.state.ViewState = model.StateJDKConfigModal
			m.state.FocusedInput = model.JdkName
			m.state.JDKStep = model.StepSelectJDKAction
			m.state.Inputs[model.JdkName].SetValue("")
			m.state.Inputs[model.JdkPath].SetValue("")
			m.state.SelectedMenuIndex = 0
			return m, nil

		case 2:
			m.state.ViewState = model.StateMavenConfigModal
			m.state.FocusedInput = model.MvnName
			m.state.MavenStep = model.StepSelectMvnAction
			m.state.Inputs[model.MvnName].SetValue("")
			m.state.Inputs[model.MvnPath].SetValue("")
			m.state.SelectedMenuIndex = 0
			return m, nil

		case 3:
			m.state.ViewState = model.StateShortcutConfigurationModal
			m.state.FocusedInput = model.FuzzyKey
			m.loadShortcutsOnInputs()
			m.state.Inputs[model.FuzzyKey].Focus()
			// Save dynamic values safely across memory pointers
			m.state.SelectedMenuIndex = 0
			return m, nil
		}
	}
	return m, nil
}
