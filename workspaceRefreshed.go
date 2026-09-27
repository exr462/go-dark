package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) workspaceRefreshed(msg config.WorkspaceRefreshedMsg) (tea.Model, tea.Cmd) {
	m.ui.Files = msg.Files
	if m.ui.SelectedProject >= 0 && m.ui.SelectedProject < len(m.ui.Config.Projects) {
		proj := m.ui.Config.Projects[m.ui.SelectedProject]
		m.ui.TreeNodes = []model.FileNode{}
		if _, err := os.Stat(proj.Path); err == nil {
			m.buildTreeNodes(proj.Path, 0)
		}
	}
	return m, func() tea.Msg { return model.FileLoadMsg("sync") }
}
