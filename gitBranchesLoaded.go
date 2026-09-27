package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
)

func (m *appModel) gitBranchesLoaded(msg config.GitBranchesLoadedMsg) (tea.Model, tea.Cmd) {
	m.ui.AvailableBranches = msg
	m.ui.SelectedGitBranch = 0
	return m, nil
}
