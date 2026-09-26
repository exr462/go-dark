package main

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
)

func (m *appModel) updateWorkspaceFiles() tea.Cmd {
	if len(m.state.Config.Projects) == 0 {
		return nil
	}

	idx := m.state.SelectedProject
	if idx < 0 || idx >= len(m.state.Config.Projects) {
		return nil
	}

	proj := m.state.Config.Projects[idx]
	fullPath := proj.Path

	return func() tea.Msg {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return config.WorkspaceRefreshedMsg{Files: []string{}}
		}

		var projectFiles []string
		for _, e := range entries {
			if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				projectFiles = append(projectFiles, e.Name())
			}
		}

		return config.WorkspaceRefreshedMsg{
			Files: projectFiles,
		}
	}
}
