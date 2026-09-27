package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateJDKModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state.JDKStep {
	case model.StepSelectJDKAction:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape):
			m.state.ViewState = m.state.PreviousViewState
			return m, nil
		case "up", "k":
			if m.state.SelectedMenuIndex > 0 {
				m.state.SelectedMenuIndex--
			}
		case "down", "j":
			if m.state.SelectedMenuIndex < 1 {
				m.state.SelectedMenuIndex++
			}
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
			if m.state.SelectedMenuIndex == 0 {
				m.state.JDKStep = model.StepAddNewJDKVersion
				m.initFields(model.JdkName, model.JdkPath)
			} else {
				m.state.JDKStep = model.StepAssignJDKToProject
				m.state.SelectedJDKIndex = 0
			}
		}
	case model.StepAddNewJDKVersion:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape):
			m.state.JDKStep = model.StepSelectJDKAction
			return m, nil
		case "tab", "down":
			m.switcheroo(model.JdkName, model.JdkPath)
		case "shift+tab", "up":
			m.switcheroo(model.JdkName, model.JdkPath)
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
			jName, jPath := m.getValues(model.JdkName, model.JdkPath)
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
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape):
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
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
			if len(m.state.Config.JDKs) > 0 && len(m.state.Config.Projects) > 0 {
				chosenJDK := m.state.Config.JDKs[m.state.SelectedJDKIndex]
				m.state.Config.Projects[m.state.SelectedProject].JDKName = chosenJDK.Name
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Assigned JDK Environment: %s", chosenJDK.Name)
			}
			m.state.ViewState = m.state.PreviousViewState
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}
