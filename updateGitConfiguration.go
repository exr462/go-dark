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
	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Escape):
		log.Printf("updateGitConfiguration Previous View State: %v", m.ui.PreviousViewState)
		m.ui.ViewState = m.ui.PreviousViewState
		return m, nil

	case "tab", "down":
		m.ui.Inputs[m.ui.FocusedInput].Blur()
		m.ui.FocusedInput = (m.ui.FocusedInput + 1) % 4
		m.ui.Inputs[m.ui.FocusedInput].Focus()
		return m, nil

	case "shift+tab", "up":
		m.ui.Inputs[m.ui.FocusedInput].Blur()
		m.ui.FocusedInput--
		if m.ui.FocusedInput < 0 {
			m.ui.FocusedInput = 3 // 🎯 FIX: Wrap cleanly to index 3 (matching your 4 total inputs)
		}
		m.ui.Inputs[m.ui.FocusedInput].Focus()
		return m, nil

	case action.GetShortcutKeyBinding(m.ui.Config.ShortCuts, action.Save):
		m.ui.Config.BasePath = m.ui.Inputs[model.GitWorkspace].Value()
		m.ui.Config.GitUsername = m.ui.Inputs[model.GitUsername].Value()
		m.ui.Config.GitEmail = m.ui.Inputs[model.GitEmail].Value()
		m.ui.Config.MaxListTag, _ = strconv.Atoi(m.ui.Inputs[model.GitMaxTagListSize].Value())
		if m.ui.Config.BasePath == "" {
			return m, nil
		}
		m.ui.Config.Projects = make([]config.Project, 0)

		for _, project := range config.AvailableProjects {
			m.ui.Config.Projects = append(m.ui.Config.Projects, config.Project{
				Deployable: project.Deployable,
				Name:       project.Name,
				// TODO fix this to be magical
				Type:   "java",
				Path:   config.ResolvePath(m.ui.Config.BasePath, project.Name),
				GitURL: fmt.Sprintf("%s%s.git", config.BaseGitURL, project.Name),
			})
		}

		_ = config.SaveConfig(m.ui.Config)
		m.ui.ViewState = m.ui.PreviousViewState
		return m, nil
	}

	var cmd tea.Cmd
	m.ui.Inputs[m.ui.FocusedInput], cmd = m.ui.Inputs[m.ui.FocusedInput].Update(msg)
	return m, cmd
}
