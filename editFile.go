package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/ui/components"
)

func (m *appModel) editFile(msg components.EditFileMsg) (tea.Model, tea.Cmd) {
	m.state.ViewState = model.StateDashboard
	if msg.Err != nil {
		m.state.LastError = msg.Err
		m.state.StatusMsg = fmt.Sprintf("❌ Editor: %v", msg.Err)
		return m, nil
	}
	m.state.ActiveCodeBuffer = msg.Content
	return m, nil
}
