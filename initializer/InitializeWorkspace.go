package initializer

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

type initializerWorkspace struct {
	ui *model.UI
}

func (m *initializerWorkspace) OnAction() tea.Cmd {
	if len(m.ui.Config.Projects) == 0 {
		return nil
	}

	idx := m.ui.SelectedProject
	if idx < 0 || idx >= len(m.ui.Config.Projects) {
		return nil
	}

	proj := m.ui.Config.Projects[idx]
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

func InitializeWorkspace(ui *model.UI) Initializer {
	return &initializerWorkspace{ui}
}
