package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
)

func (m *appModel) editFile(msg components.EditFileMsg) (tea.Model, tea.Cmd) {
	m.ui.ViewState = model.StateDashboard
	if msg.Err != nil {
		m.ui.LastError = msg.Err
		m.ui.StatusMsg = fmt.Sprintf("❌ Editor: %v", msg.Err)
		return m, nil
	}
	m.ui.ActiveCodeBuffer = msg.Content
	return m, nil
}
