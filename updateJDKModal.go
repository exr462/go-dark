package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateJDKModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state.JDKStep {
	case model.StepSelectJDKAction:
		switch msg.String() {
		case "esc":
			m.state.ViewState = model.StateConfigDeckModal
			return m, nil
		case "up", "k":
			if m.state.SelectedMenuIndex > 0 {
				m.state.SelectedMenuIndex--
			}
		case "down", "j":
			if m.state.SelectedMenuIndex < 1 {
				m.state.SelectedMenuIndex++
			}
		case "enter":
			if m.state.SelectedMenuIndex == 0 {
				m.state.JDKStep = model.StepAddNewJDKVersion
				m.state.FocusedInput = 7
				m.state.Inputs[8].SetValue("")
				m.state.Inputs[9].SetValue("")
				m.state.Inputs[8].Focus()
			} else {
				m.state.JDKStep = model.StepAssignJDKToProject
				m.state.SelectedJDKIndex = 0
			}
		}
	case model.StepAddNewJDKVersion:
		switch msg.String() {
		case "esc":
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		case "tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 16 - m.state.FocusedInput
			if m.state.FocusedInput < 8 || m.state.FocusedInput > 9 {
				m.state.FocusedInput = 8
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "shift+tab", "up":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 16 - m.state.FocusedInput
			if m.state.FocusedInput < 8 || m.state.FocusedInput > 9 {
				m.state.FocusedInput = 9
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "enter":
			jName := m.state.Inputs[8].Value()
			jPath := m.state.Inputs[7].Value()
			if jName != "" && jPath != "" {
				m.state.Config.JDKs = append(m.state.Config.JDKs, config.Profile{Name: jName, Path: jPath})
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Added Java Profile: %s", jName)
			}
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		}
		var cmd tea.Cmd
		m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
		return m, cmd
	case model.StepAssignJDKToProject:
		switch msg.String() {
		case "esc":
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		case "up", "k":
			if m.state.SelectedJDKIndex > 0 {
				m.state.SelectedJDKIndex--
			}
		case "down", "j":
			if m.state.SelectedJDKIndex < len(m.state.Config.JDKs)-1 {
				m.state.SelectedJDKIndex++
			}
		case "enter":
			if len(m.state.Config.JDKs) > 0 && len(m.state.Config.Projects) > 0 {
				chosenJDK := m.state.Config.JDKs[m.state.SelectedJDKIndex]
				m.state.Config.Projects[m.state.SelectedProject].JDKName = chosenJDK.Name
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Assigned JDK Environment: %s", chosenJDK.Name)
			}
			m.state.ViewState = model.StateDashboard
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}
