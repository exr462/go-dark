package main

import (
	"fmt"
	"log"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/action"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

//goland:noinspection GoMixedReceiverTypes
func (m *appModel) updateGitConfiguration(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Escape):
		log.Printf("updateGitConfiguration Previous View State: %v", m.state.PreviousViewState)
		m.state.ViewState = m.state.PreviousViewState
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
			m.state.FocusedInput = 3 // 🎯 FIX: Wrap cleanly to index 3 (matching your 4 total inputs)
		}
		m.state.Inputs[m.state.FocusedInput].Focus()
		return m, nil

	case action.GetShortcutKeyBinding(m.state.Config.ShortCuts, action.Save):
		m.state.Config.BasePath = m.state.Inputs[model.GitWorkspace].Value()
		m.state.Config.GitUsername = m.state.Inputs[model.GitUsername].Value()
		m.state.Config.GitEmail = m.state.Inputs[model.GitEmail].Value()
		m.state.Config.MaxListTag, _ = strconv.Atoi(m.state.Inputs[model.GitMaxTagListSize].Value())
		if m.state.Config.BasePath == "" {
			return m, nil
		}
		m.state.Config.Projects = make([]config.Project, 0)

		for _, project := range config.AvailableProjects {
			m.state.Config.Projects = append(m.state.Config.Projects, config.Project{
				Deployable: project.Deployable,
				Name:       project.Name,
				// TODO fix this to be magical
				Type:   "java",
				Path:   config.ResolvePath(m.state.Config.BasePath, project.Name),
				GitURL: fmt.Sprintf("%s%s.git", config.BaseGitURL, project.Name),
			})
		}

		_ = config.SaveConfig(m.state.Config)
		m.state.ViewState = m.state.PreviousViewState
		return m, nil
	}

	var cmd tea.Cmd
	m.state.Inputs[m.state.FocusedInput], cmd = m.state.Inputs[m.state.FocusedInput].Update(msg)
	return m, cmd
}
