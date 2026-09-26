package main

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateConfigDeckModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
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

	case "enter":
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
			m.state.JDKStep = model.StepSelectJDKAction
			m.state.SelectedMenuIndex = 0
			return m, nil

		case 2:
			m.state.ViewState = model.StateMavenConfigModal
			m.state.MavenStep = model.StepSelectMvnAction
			m.state.SelectedMenuIndex = 0
			return m, nil

		case 3:
			m.state.ViewState = model.StateAddProjectModal
			m.state.FocusedInput = model.ProjectName
			m.state.Inputs[model.ProjectName].SetValue("")
			m.state.Inputs[model.RelativeFolder].SetValue("")
			m.state.Inputs[model.StackType].SetValue("")
			m.state.Inputs[model.GitCloneURL].SetValue("")
			m.state.Inputs[model.ProjectName].Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}
