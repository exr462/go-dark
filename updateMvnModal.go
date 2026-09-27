package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateMvnModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	//goland:noinspection DuplicatedCode
	switch m.state.MavenStep {
	case model.StepSelectMvnAction:
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
				m.state.MavenStep = model.StepAddNewMvnVersion
				m.initFields(model.MvnName, model.MvnPath)
			} else {
				m.state.MavenStep = model.StepAssignMvnToProject
				m.state.SelectedMavenIndex = 0
			}
		}
	case model.StepAddNewMvnVersion:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape):
			m.state.MavenStep = model.StepSelectMvnAction
			return m, nil
		case "tab", "down":
			m.switcheroo(model.MvnName, model.MvnPath)
		case "shift+tab", "up":
			m.switcheroo(model.MvnName, model.MvnPath)
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
			mVnName, mVnPath := m.getValues(model.MvnName, model.MvnPath)
			if mVnName != "" && mVnPath != "" {
				m.state.Config.Mavens = append(m.state.Config.Mavens, config.Profile{Name: mVnName, Path: mVnPath})
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Added Maven Profile: %s", mVnName)
			}
			m.state.MavenStep = model.StepSelectMvnAction
			return m, nil
		}
		var cmd tea.Cmd
		m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
		return m, cmd
	case model.StepAssignMvnToProject:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape):
			m.state.MavenStep = model.StepSelectMvnAction
			return m, nil
		case "up", "k":
			if m.state.SelectedMavenIndex > 0 {
				m.state.SelectedMavenIndex--
			}
		case "down", "j":
			if m.state.SelectedMavenIndex < len(m.state.Config.Mavens)-1 {
				m.state.SelectedMavenIndex++
			}
		case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
			if len(m.state.Config.Mavens) > 0 && len(m.state.Config.Projects) > 0 {
				chosenMvn := m.state.Config.Mavens[m.state.SelectedMavenIndex]
				m.state.Config.Projects[m.state.SelectedProject].MavenName = chosenMvn.Name
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Assigned Maven Profile: %s", chosenMvn.Name)
			}
			m.state.ViewState = m.state.PreviousViewState
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}

func (m *appModel) initFields(name model.InputField, path model.InputField) {
	m.state.FocusedInput = name
	m.state.Inputs[name].SetValue("")
	m.state.Inputs[path].SetValue("")
	m.state.Inputs[name].Focus()
}

func (m *appModel) switcheroo(name model.InputField, path model.InputField) {
	m.state.Inputs[m.state.FocusedInput].Blur()
	if m.state.FocusedInput == name {
		m.state.FocusedInput = path
	} else if m.state.FocusedInput == path {
		m.state.FocusedInput = name
	}
	m.state.Inputs[m.state.FocusedInput].Focus()
}

func (m *appModel) getValues(name model.InputField, path model.InputField) (string, string) {
	mVnName := m.state.Inputs[name].Value()
	mVnPath := m.state.Inputs[path].Value()
	return mVnName, mVnPath
}
