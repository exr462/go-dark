package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/config"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
)

func (m *appModel) gitCheckoutComplete(msg config.GitCheckoutCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.ui.StatusMsg = fmt.Sprintf("❌ %v", msg.Err)
	} else {
		m.ui.StatusMsg = fmt.Sprintf("✅ %s", strings.TrimSpace(msg.Output))
		// Refresh fetched status across projects
		for i := range m.ui.Config.Projects {
			gitDir := filepath.Join(m.ui.Config.Projects[i].Path, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				m.ui.Config.Projects[i].Fetched = true
			}
		}
		_ = config.SaveConfig(m.ui.Config)
	}
	m.ui.ViewState = model.StateDashboard
	return m, initializer.InitializeWorkspace(m.ui).OnAction()
}
