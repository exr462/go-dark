package main

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes,GoMixedReceiverTypes,GoMixedReceiverTypes,GoMixedReceiverTypes
func (m *appModel) updateInstaller(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state.GitMissing {
		return m, nil
	}

	if m.state.InstallerStep == model.StepSetGlobalPrefs {
		switch msg.String() {
		case "esc":
			m.state.ViewState = model.StateDashboard
			return m, nil
		case "tab", "down":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput = (m.state.FocusedInput + 1) % 4
			m.state.Inputs[m.state.FocusedInput].Focus()
			return m, nil
		case "shift+tab", "up":
			m.state.Inputs[m.state.FocusedInput].Blur()
			m.state.FocusedInput--
			if m.state.FocusedInput < 0 {
				m.state.FocusedInput = 2
			}
			m.state.Inputs[m.state.FocusedInput].Focus()
			return m, nil
		case "enter":
			m.state.Config.BasePath = m.state.Inputs[model.GitWorkspace].Value()
			m.state.Config.GitUsername = m.state.Inputs[model.GitUsername].Value()
			m.state.Config.GitEmail = m.state.Inputs[model.GitEmail].Value()
			m.state.Config.MaxListTag, _ = strconv.Atoi(m.state.Inputs[model.GitMaxTagListSize].Value())
			if m.state.Config.BasePath == "" {
				return m, nil
			}
			_ = config.SaveConfig(m.state.Config)
			m.state.InstallerStep = model.StepAddFirstProject
			m.state.FocusedInput = model.ProjectName
			m.state.Inputs[model.ProjectName].Focus()
			return m, nil
		}
		var cmd tea.Cmd
		m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.state.ViewState = model.StateDashboard
		return m, nil
	case "tab", "down":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput++
		if m.state.FocusedInput > model.GitCloneURL {
			m.state.FocusedInput = model.ProjectName
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "shift+tab", "up":
		m.state.Inputs[m.state.FocusedInput].Blur()
		m.state.FocusedInput--
		if m.state.FocusedInput < model.ProjectName {
			m.state.FocusedInput = model.GitCloneURL
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
	case "enter":
		name := m.state.Inputs[model.ProjectName].Value()
		path := m.state.Inputs[model.RelativeFolder].Value()
		pType := m.state.Inputs[model.StackType].Value()
		gitURL := m.state.Inputs[model.GitCloneURL].Value()
		if name != "" && path != "" {
			m.state.Config.Projects = append(m.state.Config.Projects, config.Project{
				Name:   name,
				Path:   config.ResolvePath(m.state.Config.BasePath, path),
				Type:   strings.ToLower(pType),
				GitURL: gitURL,
			})
		}
		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = model.StateDashboard
		return m, m.updateWorkspaceFiles()
	}
	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
