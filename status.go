package main

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) status(msg model.StatusMsg) (tea.Model, tea.Cmd) {
	m.state.StatusMsg = string(msg)
	if strings.Contains(m.state.StatusMsg, "successfully completed") {
		for i := range m.state.Config.Projects {
			gitDir := filepath.Join(m.state.Config.Projects[i].Path, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				m.state.Config.Projects[i].Fetched = true
			}
		}
		_ = config.SaveConfig(m.state.Config)
		return m, m.updateWorkspaceFiles()
	}
	return m, nil
}
