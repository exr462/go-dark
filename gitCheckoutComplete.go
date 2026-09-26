package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) gitCheckoutComplete(msg config.GitCheckoutCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.state.StatusMsg = fmt.Sprintf("❌ %v", msg.Err)
	} else {
		m.state.StatusMsg = fmt.Sprintf("✅ %s", strings.TrimSpace(msg.Output))
		// Refresh fetched status across projects
		for i := range m.state.Config.Projects {
			gitDir := filepath.Join(m.state.Config.Projects[i].Path, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				m.state.Config.Projects[i].Fetched = true
			}
		}
		_ = config.SaveConfig(m.state.Config)
	}
	m.state.ViewState = model.StateDashboard
	return m, m.updateWorkspaceFiles()
}
