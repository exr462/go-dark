package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/exr462/go-dark/model"
	"github.com/exr462/go-dark/task"
)

func (m *appModel) pipelineComplete(msg task.PipelineCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		m.state.StatusMsg = "🎉 All workspace modules built successfully!"
	} else {
		m.state.StatusMsg = "❌ Pipeline compilation aborted due to build errors."
	}
	m.state.ViewState = model.StateDashboard
	return m, m.updateWorkspaceFiles()
}
