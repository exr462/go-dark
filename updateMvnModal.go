package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateMvnModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state.MavenStep {
	case model.StepSelectMvnAction:
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
				m.state.MavenStep = model.StepAddNewMvnVersion
				m.state.FocusedInput = 9
				m.state.Inputs[10].SetValue("")
				m.state.Inputs[11].SetValue("")
				m.state.Inputs[10].Focus()
			} else {
				m.state.MavenStep = model.StepAssignMvnToProject
				m.state.SelectedMavenIndex = 0
			}
		}
	case model.StepAddNewMvnVersion:
		switch msg.String() {
		case "esc":
			m.state.MavenStep = model.StepSelectMvnAction
			return m, nil
		case "tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 20 - m.state.FocusedInput
			if m.state.FocusedInput < 10 || m.state.FocusedInput > 11 {
				m.state.FocusedInput = 10
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "shift+tab", "up":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = 20 - m.state.FocusedInput
			if m.state.FocusedInput < 10 || m.state.FocusedInput > 11 {
				m.state.FocusedInput = 11
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
		case "enter":
			mVnName := m.state.Inputs[10].Value()
			mVnPath := m.state.Inputs[11].Value()
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
		case "esc":
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
		case "enter":
			if len(m.state.Config.Mavens) > 0 && len(m.state.Config.Projects) > 0 {
				chosenMvn := m.state.Config.Mavens[m.state.SelectedMavenIndex]
				m.state.Config.Projects[m.state.SelectedProject].MavenName = chosenMvn.Name
				_ = config.SaveConfig(m.state.Config)
				m.state.StatusMsg = fmt.Sprintf("✅ Assigned Maven Profile: %s", chosenMvn.Name)
			}
			m.state.ViewState = model.StateDashboard
			return m, m.updateWorkspaceFiles()
		}
	}
	return m, nil
}
