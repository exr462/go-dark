package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateMvnModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	//goland:noinspection DuplicatedCode
	switch m.ui.MavenStep {
	case model.StepSelectMvnAction:
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
				m.ui.MavenStep = model.StepAddNewMvnVersion
				m.initFields(model.MvnName, model.MvnPath)
			} else {
				m.ui.MavenStep = model.StepAssignMvnToProject
				m.ui.SelectedMavenIndex = 0
			}
		}
	case model.StepAddNewMvnVersion:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
			m.ui.MavenStep = model.StepSelectMvnAction
			return m, nil
		case "tab", "down":
			m.switcheroo(model.MvnName, model.MvnPath)
		case "shift+tab", "up":
			m.switcheroo(model.MvnName, model.MvnPath)
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
			mvnName, mvnPath := m.getValues(model.MvnName, model.MvnPath)
			if mvnName != "" && mvnPath != "" {
				m.ui.Config.Mavens = append(m.ui.Config.Mavens, config.Profile{Name: mvnName, Path: mvnPath})
				_ = config.SaveConfig(m.ui.Config)
				m.ui.StatusMsg = fmt.Sprintf("✅ Added Maven Profile: %s", mvnName)
			}
			m.ui.MavenStep = model.StepSelectMvnAction
			return m, nil
		}
		var cmd tea.Cmd
		m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
		return m, cmd
	case model.StepAssignMvnToProject:
		switch msg.String() {
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
			m.ui.MavenStep = model.StepSelectMvnAction
			return m, nil
		case "up", "k":
			if m.ui.SelectedMavenIndex > 0 {
				m.ui.SelectedMavenIndex--
			}
		case "down", "j":
			if m.ui.SelectedMavenIndex < len(m.ui.Config.Mavens)-1 {
				m.ui.SelectedMavenIndex++
			}
		case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
			if len(m.ui.Config.Mavens) > 0 && len(m.ui.Config.Projects) > 0 {
				chosenMvn := m.ui.Config.Mavens[m.ui.SelectedMavenIndex]
				m.ui.Config.Projects[m.ui.SelectedProject].MavenName = chosenMvn.Name
				_ = config.SaveConfig(m.ui.Config)
				m.ui.StatusMsg = fmt.Sprintf("✅ Assigned Maven Profile: %s", chosenMvn.Name)
			}
			m.ui.ViewState = m.ui.PreviousViewState
			return m, initializer.InitializeWorkspace(m.ui).OnAction()
		}
	}
	return m, nil
}

func (m *appModel) initFields(name model.InputField, path model.InputField) {
	m.ui.FocusedInput = name
	m.ui.Inputs[name].SetValue("")
	m.ui.Inputs[path].SetValue("")
	m.ui.Inputs[name].Focus()
}

func (m *appModel) switcheroo(name model.InputField, path model.InputField) {
	m.ui.Inputs[m.ui.FocusedInput].Blur()
	if m.ui.FocusedInput == name {
		m.ui.FocusedInput = path
	} else if m.ui.FocusedInput == path {
		m.ui.FocusedInput = name
	}
	m.ui.Inputs[m.ui.FocusedInput].Focus()
}

func (m *appModel) getValues(name model.InputField, path model.InputField) (string, string) {
	mVnName := m.ui.Inputs[name].Value()
	mVnPath := m.ui.Inputs[path].Value()
	return mVnName, mVnPath
}
