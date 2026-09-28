package main

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) status(msg model.StatusMsg) (tea.Model, tea.Cmd) {
	m.ui.StatusMsg = string(msg)
	if strings.Contains(m.ui.StatusMsg, "successfully completed") {
		for i := range m.ui.Config.Projects {
			gitDir := filepath.Join(m.ui.Config.Projects[i].Path, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				m.ui.Config.Projects[i].Fetched = true
			}
		}
		_ = config.SaveConfig(m.ui.Config)
		return m, initializer.InitializeWorkspace(m.ui).OnAction()
	}
	return m, nil
}
