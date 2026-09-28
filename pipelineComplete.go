package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/initializer"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/task"
)

func (m *appModel) pipelineComplete(msg task.PipelineCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		m.ui.StatusMsg = "🎉 All workspace modules built successfully!"
	} else {
		m.ui.StatusMsg = "❌ Pipeline compilation aborted due to build errors."
	}
	m.ui.ViewState = model.StateDashboard
	return m, initializer.InitializeWorkspace(m.ui).OnAction()
}
