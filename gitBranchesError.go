package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
)

func (m *appModel) gitBranchesError(msg config.GitBranchesErrorMsg) (tea.Model, tea.Cmd) {
	m.ui.AvailableBranches = []string{"main"}
	m.ui.SelectedGitBranch = 0
	m.ui.StatusMsg = fmt.Sprintf("❌ Git: %v", msg)
	return m, nil
}
