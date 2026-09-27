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
	switch m.ui.JDKStep {
	case model.StepSelectJDKAction:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
			m.ui.ViewState = m.ui.PreviousViewState
			return m, nil
		case "up", "k":
			if m.ui.SelectedMenuIndex > 0 {
				m.ui.SelectedMenuIndex--
			}
		case "down", "j":
			if m.ui.SelectedMenuIndex < 1 {
				m.ui.SelectedMenuIndex++
			}
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
			if m.ui.SelectedMenuIndex == 0 {
				m.ui.JDKStep = model.StepAddNewJDKVersion
				m.initFields(model.JdkName, model.JdkPath)
			} else {
				m.ui.JDKStep = model.StepAssignJDKToProject
				m.ui.SelectedJDKIndex = 0
			}
		}
	case model.StepAddNewJDKVersion:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
			m.ui.JDKStep = model.StepSelectJDKAction
			return m, nil
		case "tab", "down":
			m.switcheroo(model.JdkName, model.JdkPath)
		case "shift+tab", "up":
			m.switcheroo(model.JdkName, model.JdkPath)
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
			jName, jPath := m.getValues(model.JdkName, model.JdkPath)
			if jName != "" && jPath != "" {
				m.ui.Config.JDKs = append(m.ui.Config.JDKs, config.Profile{Name: jName, Path: jPath})
				_ = config.SaveConfig(m.ui.Config)
				m.ui.StatusMsg = fmt.Sprintf("✅ Added Java Profile: %s", jName)
			}
			m.ui.JDKStep = model.StepSelectJDKAction
			return m, nil
		}
		var cmd tea.Cmd
		m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
		return m, cmd
	case model.StepAssignJDKToProject:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
			m.ui.JDKStep = model.StepSelectJDKAction
			return m, nil
		case "up", "k":
			if m.ui.SelectedJDKIndex > 0 {
				m.ui.SelectedJDKIndex--
			}
		case "down", "j":
			if m.ui.SelectedJDKIndex < len(m.ui.Config.JDKs)-1 {
				m.ui.SelectedJDKIndex++
			}
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
			if len(m.ui.Config.JDKs) > 0 && len(m.ui.Config.Projects) > 0 {
				chosenJDK := m.ui.Config.JDKs[m.ui.SelectedJDKIndex]
				m.ui.Config.Projects[m.ui.SelectedProject].JDKName = chosenJDK.Name
				_ = config.SaveConfig(m.ui.Config)
				m.ui.StatusMsg = fmt.Sprintf("✅ Assigned JDK Environment: %s", chosenJDK.Name)
			}
			m.ui.ViewState = m.ui.PreviousViewState
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}
