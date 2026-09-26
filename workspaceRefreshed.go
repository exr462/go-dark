package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) workspaceRefreshed(msg config.WorkspaceRefreshedMsg) (tea.Model, tea.Cmd) {
	m.state.Files = msg.Files
	if m.state.SelectedProject >= 0 && m.state.SelectedProject < len(m.state.Config.Projects) {
		proj := m.state.Config.Projects[m.state.SelectedProject]
		m.state.TreeNodes = []model.FileNode{}
		if _, err := os.Stat(proj.Path); err == nil {
			m.buildTreeNodes(proj.Path, 0)
		}
	}
	return m, func() tea.Msg { return model.FileLoadMsg("sync") }
}
